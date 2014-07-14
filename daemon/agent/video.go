package agent

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"log"
)

func init() {
	rendererConstructors["videoRenderAgent"] = newVideoRenderer
}

type videoRenderer struct {
	zencoderNotificationUrl string
	zencoderS3Bucket        string
}

func (renderer *videoRenderer) generateTemplates(zencoderS3Bucket string) {
	localTemplates := []*common.Template{
		&common.Template{
			Id:          common.VideoConversionTemplateId,
			RenderAgent: "videoRenderAgent",
			Group:       "7A96",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeOutput, []string{"m3u8"}},
				// TODO[JSH]: Load this from config/define it not in the template
				common.Attribute{"forceS3Location", []string{renderer.zencoderS3Bucket}},
			},
		},
	}
	templates = append(templates, localTemplates...)
}

func newVideoRenderer(params map[string]string) Renderer {
	renderer := new(videoRenderer)
	var ok bool
	renderer.zencoderNotificationUrl, ok = params["zencoderNotificationUrl"]
	if !ok {
		log.Fatal("Missing zencoderNotificationUrl parameter from videoRenderAgent")
	}
	renderer.zencoderS3Bucket, ok = params["zencoderS3Bucket"]
	if !ok {
		log.Fatal("Missing zencoderS3Bucket parameter from videoRenderAgent")
	}

	renderer.generateTemplates(renderer.zencoderS3Bucket)

	return renderer
}

func (renderer *videoRenderer) renderGeneratedAsset(id string, renderAgent *genericRenderAgent) {
	renderAgent.metrics.workProcessed.Mark(1)

	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Println("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	surl := fmt.Sprintf("%s/%s.m3u8", generatedAsset.Location, id)
	generatedAsset.AddAttribute("streamingUrl", []string{common.S3ToHttps(surl)})
	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)
	sourceAsset, err := renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := getSourceAssetFileType(sourceAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileType), nil}
		return
	}
	renderAgent.metrics.fileTypeCount[fileType].Inc(1)

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	input := urls[0]

	templates, err := renderAgent.templateManager.FindByIds([]string{generatedAsset.TemplateId})
	if err != nil || !templates[0].HasAttribute("zencoderNotificationUrl") {
		log.Println("Could not retrieve notification URL from template")
		return
	}
	// Zencoder will put the files the folder generatedAsset.Location
	// The filename for the HLS playlist will be generatedAsset.Id with .m3u8 extension
	settings := common.BuildZencoderSettings(input, generatedAsset.Location, generatedAsset.Id, renderer.zencoderNotificationUrl)
	arr, _ := json.MarshalIndent(settings, "", "	")
	log.Println(string(arr))
	job, err := renderAgent.agentManager.zencoder.CreateJob(settings)
	if err != nil {
		log.Println("Zencoder error:", err)
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}
	log.Println("Created Zencoder job", job)

	// The webhook API will mark the GA as completed once Zencoder sends back a notification
	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusDelegated, nil}
}
