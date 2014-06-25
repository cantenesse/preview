package render

import (
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/rcrowley/go-metrics"
	"log"
	"time"
)

func init() {
	Renderers = make(map[string]rendererConstructor)
}

var Renderers map[string]rendererConstructor

type RenderAgentWorkChannel chan string

type RenderStatusChannel chan RenderStatus

type RenderStatus struct {
	GeneratedAssetId string
	Status           string
	Service          string
}

type generatedAssetUpdate struct {
	status     string
	attributes []common.Attribute
}

type Renderer interface {
	renderGeneratedAsset(id string)
}

type rendererConstructor func(*genericRenderAgent, map[string]string) Renderer

type RenderAgentMetrics struct {
	workProcessed metrics.Meter
	convertTime   metrics.Timer
	fileTypeCount map[string]metrics.Counter
}

type genericRenderAgent struct {
	name                 string
	renderer             Renderer
	metrics              *RenderAgentMetrics
	sasm                 common.SourceAssetStorageManager
	gasm                 common.GeneratedAssetStorageManager
	templateManager      common.TemplateManager
	downloader           common.Downloader
	uploader             common.Uploader
	workChannel          RenderAgentWorkChannel
	statusListeners      []RenderStatusChannel
	temporaryFileManager common.TemporaryFileManager
	agentManager         *RenderAgentManager
	stop                 chan (chan bool)
}

func newRenderAgentMetrics(registry metrics.Registry, name string, supportedFileTypes []string) *RenderAgentMetrics {
	agentMetrics := new(RenderAgentMetrics)
	agentMetrics.workProcessed = metrics.NewMeter()
	agentMetrics.convertTime = metrics.NewTimer()

	agentMetrics.fileTypeCount = make(map[string]metrics.Counter)

	for _, filetype := range supportedFileTypes {
		agentMetrics.fileTypeCount[filetype] = metrics.NewCounter()
		registry.Register(fmt.Sprintf("%s.%sCount", name, filetype), agentMetrics.fileTypeCount[filetype])
	}

	registry.Register(fmt.Sprintf("%s.workProcessed", name), agentMetrics.workProcessed)
	registry.Register(fmt.Sprintf("%s.convertTime", name), agentMetrics.convertTime)

	return agentMetrics
}

func newGenericRenderAgent(
	name string,
	params map[string]string,
	metrics *RenderAgentMetrics,
	agentManager *RenderAgentManager,
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	downloader common.Downloader,
	uploader common.Uploader,
	workChannel RenderAgentWorkChannel) *genericRenderAgent {

	renderAgent := new(genericRenderAgent)
	renderAgent.name = name
	renderAgent.metrics = metrics
	renderAgent.agentManager = agentManager
	renderAgent.sasm = sasm
	renderAgent.gasm = gasm
	renderAgent.templateManager = templateManager
	renderAgent.temporaryFileManager = temporaryFileManager
	renderAgent.downloader = downloader
	renderAgent.uploader = uploader
	renderAgent.workChannel = workChannel
	renderAgent.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderAgent.stop = make(chan (chan bool))
	renderAgent.renderer = Renderers[name](renderAgent, params)

	go renderAgent.start()

	return renderAgent
}

func (renderAgent *genericRenderAgent) start() {
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

func (renderAgent *genericRenderAgent) Stop() {
	callback := make(chan bool)
	renderAgent.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderAgent.stop)
}

func (renderAgent *genericRenderAgent) AddStatusListener(listener RenderStatusChannel) {
	renderAgent.statusListeners = append(renderAgent.statusListeners, listener)
}

func (renderAgent *genericRenderAgent) Dispatch() RenderAgentWorkChannel {
	return renderAgent.workChannel
}

func (renderAgent *genericRenderAgent) renderGeneratedAsset(id string) {
	renderAgent.renderer.renderGeneratedAsset(id)
}

func (renderAgent *genericRenderAgent) commitStatus(id string, existingAttributes []common.Attribute) chan generatedAssetUpdate {
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
							listener <- RenderStatus{id, status, renderAgent.name}
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

func (renderAgent *genericRenderAgent) getSourceAsset(generatedAsset *common.GeneratedAsset) (*common.SourceAsset, error) {
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

func (renderAgent *genericRenderAgent) tryDownload(urls []string, source string) (common.TemporaryFile, error) {
	for _, url := range urls {
		tempFile, err := renderAgent.downloader.Download(url, source)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, common.ErrorNoDownloadUrlsWork
}
