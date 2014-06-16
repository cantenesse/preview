package render

import (
//	"bytes"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/rcrowley/go-metrics"
	"github.com/brandscreen/zencoder"
//	"io/ioutil"
	"log"
	"os"
//	"os/exec"
	"path/filepath"
//	"regexp"
//	"strconv"
//	"strings"
	"time"
	"encoding/json"
)

type videoRenderAgent struct {
	metrics              *videoRenderAgentMetrics
	sasm                 common.SourceAssetStorageManager
	gasm                 common.GeneratedAssetStorageManager
	templateManager      common.TemplateManager
	downloader           common.Downloader
	uploader             common.Uploader
	workChannel          RenderAgentWorkChannel
	statusListeners      []RenderStatusChannel
	temporaryFileManager common.TemporaryFileManager
	agentManager         *RenderAgentManager
	tempFileBasePath     string
	stop                 chan (chan bool)
	zencoder *zencoder.Zencoder
	zencoderS3Bucket string
	zencoderNotificationUrl string
}

type videoRenderAgentMetrics struct {
	workProcessed metrics.Meter
	convertTime   metrics.Timer
	docCount      metrics.Counter
	docxCount     metrics.Counter
	pptCount      metrics.Counter
	pptxCount     metrics.Counter
}

func newVideoRenderAgent(
	metrics *videoRenderAgentMetrics,
	agentManager *RenderAgentManager,
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	downloader common.Downloader,
	uploader common.Uploader,
	tempFileBasePath string,
	workChannel RenderAgentWorkChannel,
	zencoder *zencoder.Zencoder,
	s3Bucket string,
	notificationUrl string) RenderAgent {
	
	renderAgent := new(videoRenderAgent)
	renderAgent.metrics = metrics
	renderAgent.agentManager = agentManager
	renderAgent.sasm = sasm
	renderAgent.gasm = gasm
	renderAgent.templateManager = templateManager
	renderAgent.temporaryFileManager = temporaryFileManager
	renderAgent.downloader = downloader
	renderAgent.uploader = uploader
	renderAgent.workChannel = workChannel
	renderAgent.tempFileBasePath = tempFileBasePath
	renderAgent.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderAgent.stop = make(chan (chan bool))
	
	renderAgent.zencoder = zencoder
	renderAgent.zencoderS3Bucket = s3Bucket
	renderAgent.zencoderNotificationUrl = notificationUrl	
	go renderAgent.start()

	return renderAgent
}

func newVideoRenderAgentMetrics(registry metrics.Registry) *videoRenderAgentMetrics {
	videoMetrics := new(videoRenderAgentMetrics)
	videoMetrics.workProcessed = metrics.NewMeter()
	videoMetrics.convertTime = metrics.NewTimer()
	// TODO: determine file types
	// documentMetrics.pptxCount = metrics.NewCounter()

	registry.Register("videoRenderAgent.workProcessed", videoMetrics.workProcessed)
	registry.Register("videoRenderAgent.convertTime", videoMetrics.convertTime)
	// registry.Register("documentRenderAgent.pptxCount", documentMetrics.pptxCount)

	return videoMetrics
}

func (renderAgent *videoRenderAgent) start() {
	for {
		select {
		case ch, ok := <-renderAgent.stop:
			{
				log.Println("Stopping")
				if !ok {
					return
				}
				ch <- true
				return
			}
		case id, ok := <-renderAgent.workChannel:
			{
				if !ok {
					return
				}
				log.Println("Received dispatch message", id)
				renderAgent.renderGeneratedAsset(id)
			}
		}
	}
}

func (renderAgent *videoRenderAgent) Stop() {
	callback := make(chan bool)
	renderAgent.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderAgent.stop)
}

func (renderAgent *videoRenderAgent) AddStatusListener(listener RenderStatusChannel) {
	renderAgent.statusListeners = append(renderAgent.statusListeners, listener)
}

func (renderAgent *videoRenderAgent) Dispatch() RenderAgentWorkChannel {
	return renderAgent.workChannel
}

func (renderAgent *videoRenderAgent) renderGeneratedAsset(id string) {	
	renderAgent.metrics.workProcessed.Mark(1)

	// 1. Get the generated asset
	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}
	
	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)

	// 2. Get the source asset
	sourceAsset, err := renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		switch fileType {
		// TODO: Complete metrics
		/*
		case "pptx":
			renderAgent.metrics.pptxCount.Inc(1)
                */
		}
	}

	// 3. Get the template... not needed yet

	// 4. Fetch the source asset file
	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	// Dont need to download it; Zencoder downloads it for us
	// sourceFile, err := renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
	// if err != nil {
	// 	statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
	// 	return
	// }
	// defer sourceFile.Release()

	// TODO: Determine what to do here
	// 	// 5. Create a temporary destination directory.
	// destination, err := renderAgent.createTemporaryDestinationDirectory()
	// if err != nil {
	// 	statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
	// 	return
	// }
	// destinationTemporaryFile := renderAgent.temporaryFileManager.Create(destination)
	// defer destinationTemporaryFile.Release()
	input := urls[0]
	outputFileName := id

	settings := util.BuildZencoderSettings(input, "s3://" + renderAgent.zencoderS3Bucket + "/" + outputFileName, outputFileName, renderAgent.zencoderNotificationUrl)
	arr, _ := json.MarshalIndent(settings, "", "	")
	log.Println(string(arr))
	job, err := renderAgent.zencoder.CreateJob(settings)
	if err != nil {
		log.Println("Zencoder error:", err)
	}
	log.Println("Created Zencoder job", job)
	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusDelegated, nil}
}

func (renderAgent *videoRenderAgent) getSourceAsset(generatedAsset *common.GeneratedAsset) (*common.SourceAsset, error) {
	sourceAssets, err := renderAgent.sasm.FindBySourceAssetId(generatedAsset.SourceAssetId)
	if err != nil {
		return nil, err
	}
	for _, sourceAsset := range sourceAssets {
		if sourceAsset.IdType == generatedAsset.SourceAssetType {
			return sourceAsset, nil
		}
	}
	return nil, common.ErrorNoSourceAssetsFoundForId
}

func (renderAgent *videoRenderAgent) tryDownload(urls []string, source string) (common.TemporaryFile, error) {
	for _, url := range urls {
		tempFile, err := renderAgent.downloader.Download(url, source)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, common.ErrorNoDownloadUrlsWork
}

func (renderAgent *videoRenderAgent) commitStatus(id string, existingAttributes []common.Attribute) chan generatedAssetUpdate {
	commitChannel := make(chan generatedAssetUpdate, 10)

	go func() {
		status := common.NewGeneratedAssetError(common.ErrorUnknownError)
		attributes := make([]common.Attribute, 0, 0)
		for _, attribute := range existingAttributes {
			attributes = append(attributes, attribute)
		}
		for {
			select {
			case message, ok := <-commitChannel:
				{
					if !ok {
						for _, listener := range renderAgent.statusListeners {
							listener <- RenderStatus{id, status, common.RenderAgentDocument}
						}
						generatedAsset, err := renderAgent.gasm.FindById(id)
						if err != nil {
							panic(err)
							return
						}
						generatedAsset.Status = status
						generatedAsset.Attributes = attributes
						log.Println("Updating", generatedAsset)
						renderAgent.gasm.Update(generatedAsset)
						return
					}
					status = message.status
					if message.attributes != nil {
						for _, attribute := range message.attributes {
							attributes = append(attributes, attribute)
						}
					}
				}
			}
		}
	}()
	return commitChannel
}

func (renderAgent *videoRenderAgent) createTemporaryDestinationDirectory() (string, error) {
	uuid, err := util.NewUuid()
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(renderAgent.tempFileBasePath, uuid)
	err = os.MkdirAll(tmpPath, 0777)
	if err != nil {
		log.Println("error creating tmp dir", err)
		return "", err
	}
	return tmpPath, nil
}
