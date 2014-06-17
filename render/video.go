package render

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/rcrowley/go-metrics"
	"github.com/brandscreen/zencoder"
	"log"
	"time"
)

type videoRenderAgent struct {
	metrics              *videoRenderAgentMetrics
	sasm                 common.SourceAssetStorageManager
	gasm                 common.GeneratedAssetStorageManager
	templateManager      common.TemplateManager
	workChannel          RenderAgentWorkChannel
	statusListeners      []RenderStatusChannel
	agentManager         *RenderAgentManager
	stop                 chan (chan bool)
	zencoder *zencoder.Zencoder
	zencoderS3Bucket string
	zencoderNotificationUrl string
}

type videoRenderAgentMetrics struct {
	workProcessed metrics.Meter
	convertTime   metrics.Timer
	// TODO: finish metrics
	mp4Count     metrics.Counter
}

func newVideoRenderAgent(
	metrics *videoRenderAgentMetrics,
	agentManager *RenderAgentManager,
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
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
	renderAgent.workChannel = workChannel
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
	videoMetrics.mp4Count = metrics.NewCounter()

	registry.Register("videoRenderAgent.workProcessed", videoMetrics.workProcessed)
	registry.Register("videoRenderAgent.convertTime", videoMetrics.convertTime)
	registry.Register("videoRenderAgent.mp4Count", videoMetrics.mp4Count)

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

	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}
	
	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)

	sourceAsset, err := renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		switch fileType {
		// TODO: Complete metrics
		case "mp4":
			renderAgent.metrics.mp4Count.Inc(1)
		}
	}

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	input := urls[0]
	// Zencoder will put the files the folder generatedAsset.Location
	// The filename for the HLS playlist will be generatedAsset.Id with .m3u8 extension 
	settings := util.BuildZencoderSettings(input, generatedAsset.Location, generatedAsset.Id, renderAgent.zencoderNotificationUrl)
	//arr, _ := json.MarshalIndent(settings, "", "	")
	//log.Println(string(arr))
	job, err := renderAgent.zencoder.CreateJob(settings)
	if err != nil {
		log.Println("Zencoder error:", err)
	     statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
	     return
	}
	log.Println("Created Zencoder job", job)

	// The webhook API will mark the GA as completed once Zencoder sends back a notification
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
