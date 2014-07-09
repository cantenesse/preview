package agent

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"log"
)

func init() {
	renderers["videoRenderAgent"] = newVideoRenderer
	localTemplates := []*common.Template{
		&common.Template{
			Id:          common.VideoConversionTemplateId,
			RenderAgent: "videoRenderAgent",
			Group:       "7A96",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeOutput, []string{"m3u8"}},
				// TODO[JSH]: Load this from config/define it not in the template
				common.Attribute{"forceS3Location", []string{"YOUR_S3_BUCKET"}},
			},
		},
	}
	templates = append(templates, localTemplates...)
}

type videoRenderer struct {
	renderAgent             *genericRenderAgent
	zencoderNotificationUrl string
}

func newVideoRenderer(renderAgent *genericRenderAgent, params map[string]string) Renderer {
	renderer := new(videoRenderer)
	renderer.renderAgent = renderAgent
	renderer.zencoderNotificationUrl = params["zencoderNotificationUrl"]
	return renderer
}

func (renderer *videoRenderer) renderGeneratedAsset(id string) {
	renderer.renderAgent.metrics.workProcessed.Mark(1)

	generatedAsset, err := renderer.renderAgent.gasm.FindById(id)
	if err != nil {
		log.Println("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	surl := fmt.Sprintf("%s/%s.m3u8", generatedAsset.Location, id)
	generatedAsset.AddAttribute("streamingUrl", []string{common.S3ToHttps(surl)})
	statusCallback := renderer.renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderer.renderAgent.gasm.Update(generatedAsset)
	sourceAsset, err := renderer.renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := getSourceAssetFileType(sourceAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileType), nil}
		return
	}
	renderer.renderAgent.metrics.fileTypeCount[fileType].Inc(1)

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	input := urls[0]

	templates, err := renderer.renderAgent.templateManager.FindByIds([]string{generatedAsset.TemplateId})
	if err != nil || !templates[0].HasAttribute("zencoderNotificationUrl") {
		log.Println("Could not retrieve notification URL from template")
		return
	}
	// Zencoder will put the files the folder generatedAsset.Location
	// The filename for the HLS playlist will be generatedAsset.Id with .m3u8 extension
	settings := common.BuildZencoderSettings(input, generatedAsset.Location, generatedAsset.Id, renderer.zencoderNotificationUrl)
	arr, _ := json.MarshalIndent(settings, "", "	")
	log.Println(string(arr))
	job, err := renderer.renderAgent.agentManager.zencoder.CreateJob(settings)
	if err != nil {
		log.Println("Zencoder error:", err)
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}
	log.Println("Created Zencoder job", job)

	// The webhook API will mark the GA as completed once Zencoder sends back a notification
	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusDelegated, nil}
}
