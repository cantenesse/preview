package render

import (
	"encoding/json"
	"fmt"
	"github.com/jherman3/zencoder"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"log"
)

type videoRenderer struct {
	renderAgent             *genericRenderAgent
	zencoder                *zencoder.Zencoder
	zencoderS3Bucket        string
	zencoderNotificationUrl string
}

func newVideoRenderer(renderAgent *genericRenderAgent, zencoder *zencoder.Zencoder, zencoderS3Bucket, zencoderNotificationUrl string) *videoRenderer {
	renderer := new(videoRenderer)
	renderer.renderAgent = renderAgent
	renderer.zencoderS3Bucket = zencoderS3Bucket
	renderer.zencoderNotificationUrl = zencoderNotificationUrl

	return renderer
}

func (renderer *videoRenderer) renderGeneratedAsset(id string) {
	renderer.renderAgent.metrics.workProcessed.Mark(1)

	generatedAsset, err := renderer.renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	surl := fmt.Sprintf("%s/%s.m3u8", generatedAsset.Location, id)
	generatedAsset.AddAttribute("streamingUrl", []string{surl})
	statusCallback := renderer.renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderer.renderAgent.gasm.Update(generatedAsset)
	sourceAsset, err := renderer.renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if _, supports := renderer.renderAgent.metrics.fileTypeCount[fileType]; !supports {
		log.Println("VideoRenderAgent doesn't support filetype", fileType)
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	if err == nil {
		renderer.renderAgent.metrics.fileTypeCount[fileType].Inc(1)
	}

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	input := urls[0]
	// Zencoder will put the files the folder generatedAsset.Location
	// The filename for the HLS playlist will be generatedAsset.Id with .m3u8 extension
	settings := util.BuildZencoderSettings(input, generatedAsset.Location, generatedAsset.Id, renderer.zencoderNotificationUrl)
	arr, _ := json.MarshalIndent(settings, "", "	")
	log.Println(string(arr))
	job, err := renderer.zencoder.CreateJob(settings)
	if err != nil {
		log.Println("Zencoder error:", err)
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}
	log.Println("Created Zencoder job", job)

	// The webhook API will mark the GA as completed once Zencoder sends back a notification
	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusDelegated, nil}
}
