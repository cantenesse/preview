package agent

import (
	"fmt"
	"github.com/jherman3/zencoder"
	"github.com/ngerakines/preview/common"
	"github.com/rcrowley/go-metrics"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RenderAgentManager struct {
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	temporaryFileManager         common.TemporaryFileManager
	uploader                     common.Uploader
	workStatus                   RenderStatusChannel
	workChannels                 map[string]RenderAgentWorkChannel
	renderAgents                 map[string][]*genericRenderAgent
	activeWork                   map[string][]string
	maxWork                      map[string]int
	enabledRenderAgents          map[string]bool
	renderAgentCount             map[string]int
	supportedFileTypes           map[string][]string
	metrics                      map[string]*RenderAgentMetrics
	stop                         chan (chan bool)
	mu                           sync.Mutex
	// I feel like there should be a way to do this without giving RenderAgentManager a Zencoder
	zencoder *zencoder.Zencoder
}

func NewRenderAgentManager(
	registry metrics.Registry,
	sourceAssetStorageManager common.SourceAssetStorageManager,
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	uploader common.Uploader,
	workDispatcherEnabled bool,
	zencoder *zencoder.Zencoder,
	supportedFileTypes map[string][]string) *RenderAgentManager {

	agentManager := new(RenderAgentManager)
	agentManager.sourceAssetStorageManager = sourceAssetStorageManager
	agentManager.generatedAssetStorageManager = generatedAssetStorageManager
	agentManager.templateManager = templateManager
	agentManager.uploader = uploader

	agentManager.temporaryFileManager = temporaryFileManager
	agentManager.workStatus = make(RenderStatusChannel, 100)
	agentManager.workChannels = make(map[string]RenderAgentWorkChannel)
	for k := range supportedFileTypes {
		agentManager.workChannels[k] = make(RenderAgentWorkChannel, 200)
	}
	agentManager.renderAgents = make(map[string][]*genericRenderAgent)
	agentManager.activeWork = make(map[string][]string)
	agentManager.maxWork = make(map[string]int)
	agentManager.enabledRenderAgents = make(map[string]bool)
	agentManager.renderAgentCount = make(map[string]int)

	agentManager.zencoder = zencoder

	agentManager.supportedFileTypes = supportedFileTypes
	agentManager.metrics = make(map[string]*RenderAgentMetrics)
	for k, v := range supportedFileTypes {
		agentManager.metrics[k] = newRenderAgentMetrics(registry, k, v)
	}

	agentManager.stop = make(chan (chan bool))
	if workDispatcherEnabled {
		go agentManager.run()
	}

	return agentManager
}

func (agentManager *RenderAgentManager) ActiveWorkForRenderAgent(renderAgent string) (bool, int, []string) {
	activeWork, hasActiveWork := agentManager.activeWork[renderAgent]
	if hasActiveWork {
		return agentManager.isRenderAgentEnabled(renderAgent), agentManager.getRenderAgentCount(renderAgent), activeWork
	}
	return agentManager.isRenderAgentEnabled(renderAgent), agentManager.getRenderAgentCount(renderAgent), []string{}
}

func (agentManager *RenderAgentManager) SetRenderAgentInfo(name string, value bool, count int) {
	agentManager.enabledRenderAgents[name] = value
	agentManager.renderAgentCount[name] = count
}

func (agentManager *RenderAgentManager) isRenderAgentEnabled(name string) bool {
	value, hasValue := agentManager.enabledRenderAgents[name]
	if hasValue {
		return value
	}
	return false
}

func (agentManager *RenderAgentManager) getRenderAgentCount(name string) int {
	value, hasValue := agentManager.renderAgentCount[name]
	if hasValue {
		return value
	}
	return 0
}

func (agentManager *RenderAgentManager) CreateWorkFromTemplates(sourceAssetId, url string, attributes map[string][]string, templateIds []string) {
	sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	if err != nil {
		return
	}
	size, hasSize := attributes["size"]
	if hasSize {
		sourceAsset.AddAttribute(common.SourceAssetAttributeSize, size)
	}
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{url})

	fileType, hasType := attributes["type"]
	if hasType {
		sourceAsset.AddAttribute(common.SourceAssetAttributeType, fileType)
	}

	agentManager.sourceAssetStorageManager.Store(sourceAsset)

	templates, err := agentManager.templateManager.FindByIds(templateIds)
	if err != nil {
		return
	}

	status := common.DefaultGeneratedAssetStatus
	for _, template := range templates {
		var location string
		forceLoc := template.GetAttribute("forceS3Location")
		if len(forceLoc) == 1 && len(forceLoc[0]) > 0 {
			location = fmt.Sprintf("s3://%s/%s", forceLoc[0], sourceAssetId)
		} else {
			location = agentManager.uploader.Url(sourceAsset, template, 0)
		}
		ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template.Id, location)

		if err == nil {
			status, dispatchFunc := agentManager.canDispatch(ga.Id, status, template)
			if status != ga.Status {
				ga.Status = status
			}
			agentManager.generatedAssetStorageManager.Store(ga)
			if dispatchFunc != nil {
				defer dispatchFunc()
			}
		} else {
			log.Println("error creating generated asset from source asset", err)
			return
		}
	}
}

func (agentManager *RenderAgentManager) CreateWork(sourceAssetId, url, fileType string, size int64) {
	sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	if err != nil {
		return
	}
	sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{strconv.FormatInt(size, 10)})
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{url})
	sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{fileType})

	agentManager.sourceAssetStorageManager.Store(sourceAsset)

	templates, status, err := agentManager.whichRenderAgent(fileType)
	if err != nil {
		log.Println("error determining which render agent to use", err)
		return
	}

	placeholderSizes := make(map[string]string)
	for _, template := range templates {
		placeholderSize, err := common.GetFirstAttribute(template, common.TemplateAttributePlaceholderSize)
		if err == nil {
			placeholderSizes[template.Id] = placeholderSize
		}
	}

	for _, template := range templates {
		var location string
		forceLoc := template.GetAttribute("forceS3Location")
		if len(forceLoc) == 1 && len(forceLoc[0]) > 0 {
			location = fmt.Sprintf("s3://%s/%s", forceLoc[0], sourceAssetId)
		} else {
			location = agentManager.uploader.Url(sourceAsset, template, 0)
		}
		ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template.Id, location)

		if err == nil {
			status, dispatchFunc := agentManager.canDispatch(ga.Id, status, template)
			if status != ga.Status {
				ga.Status = status
			}
			agentManager.generatedAssetStorageManager.Store(ga)
			if dispatchFunc != nil {
				defer dispatchFunc()
			}
		} else {
			log.Println("error creating generated asset from source asset", err)
			return
		}
	}
}

func (agentManager *RenderAgentManager) CreateDerivedWork(derivedSourceAsset *common.SourceAsset, templates []*common.Template, firstPage int, lastPage int) error {
	placeholderSizes := make(map[string]string)
	for _, template := range templates {
		placeholderSize, err := common.GetFirstAttribute(template, common.TemplateAttributePlaceholderSize)
		if err != nil {
			return err
		}
		placeholderSizes[template.Id] = placeholderSize
	}

	for page := firstPage; page < lastPage; page++ {
		for _, template := range templates {
			location := agentManager.uploader.Url(derivedSourceAsset, template, int32(page))
			generatedAsset, err := common.NewGeneratedAssetFromSourceAsset(derivedSourceAsset, template.Id, location)
			if err == nil {
				generatedAsset.AddAttribute(common.GeneratedAssetAttributePage, []string{strconv.Itoa(page)})
				status, dispatchFunc := agentManager.canDispatch(generatedAsset.Id, generatedAsset.Status, template)
				if status != generatedAsset.Status {
					generatedAsset.Status = status
				}
				if dispatchFunc != nil {
					defer dispatchFunc()
				}
				agentManager.generatedAssetStorageManager.Store(generatedAsset)
			}
		}
	}
	return nil
}

func (agentManager *RenderAgentManager) whichRenderAgent(fileType string) ([]*common.Template, string, error) {
	fileType = strings.ToLower(fileType)
	var templateIds []string
	if common.Contains(agentManager.supportedFileTypes["documentRenderAgent"], fileType) {
		templateIds = []string{common.DocumentConversionTemplateId}
	} else if common.Contains(agentManager.supportedFileTypes["videoRenderAgent"], fileType) {
		templateIds = []string{common.VideoConversionTemplateId}
	} else if common.Contains(agentManager.supportedFileTypes["imageMagickRenderAgent"], fileType) {
		templateIds = common.LegacyDefaultTemplates
	} else {
		return nil, common.GeneratedAssetStatusFailed, common.ErrorNoRenderersSupportFileType
	}
	templates, err := agentManager.templateManager.FindByIds(templateIds)
	for _, t := range templates {
		log.Println(t.Id, t.RenderAgent)
	}
	if err != nil {
		return nil, common.GeneratedAssetStatusFailed, err
	}
	return templates, common.DefaultGeneratedAssetStatus, nil
}

func (agentManager *RenderAgentManager) canDispatch(generatedAssetId, status string, template *common.Template) (string, func()) {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()
	max, hasMax := agentManager.maxWork[template.RenderAgent]
	if !hasMax {
		return status, nil
	}
	max = max * 4
	activeWork, hasCount := agentManager.activeWork[template.RenderAgent]
	if !hasCount {
		return status, nil
	}
	if len(activeWork) >= max {
		return status, nil
	}
	renderAgents, hasRenderAgent := agentManager.renderAgents[template.RenderAgent]
	if !hasRenderAgent {
		return status, nil
	}
	if len(renderAgents) == 0 {
		return status, nil
	}
	renderAgent := renderAgents[0]
	agentManager.activeWork[template.RenderAgent] = uniqueListWith(agentManager.activeWork[template.RenderAgent], generatedAssetId)

	return common.GeneratedAssetStatusScheduled, func() {
		go func() {
			renderAgent.Dispatch() <- generatedAssetId
		}()
	}
}

func (agentManager *RenderAgentManager) AddListener(listener RenderStatusChannel) {
	for _, renderAgents := range agentManager.renderAgents {
		for _, renderAgent := range renderAgents {
			renderAgent.AddStatusListener(listener)
		}
	}
}

func (agentManager *RenderAgentManager) Stop() {
	for name, renderAgents := range agentManager.renderAgents {
		log.Println("Stopping", name+"s.")
		for _, renderAgent := range renderAgents {
			renderAgent.Stop()
		}
		log.Println("All", name+"s", "successfully stopped.")
	}
	for _, workChannel := range agentManager.workChannels {
		close(workChannel)
	}

	callback := make(chan bool)
	agentManager.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(agentManager.stop)
}

func (agentManager *RenderAgentManager) AddRenderAgent(name string, params map[string]string, downloader common.Downloader, uploader common.Uploader, maxWorkIncrease int) {
	renderAgent := newGenericRenderAgent(name, params, agentManager.metrics[name], agentManager, agentManager.sourceAssetStorageManager, agentManager.generatedAssetStorageManager, agentManager.templateManager, agentManager.temporaryFileManager, downloader, uploader, agentManager.workChannels[name])
	renderAgent.AddStatusListener(agentManager.workStatus)

	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()

	renderAgents, hasRenderAgents := agentManager.renderAgents[name]
	if !hasRenderAgents {
		renderAgents = make([]*genericRenderAgent, 0, 0)
		renderAgents = append(renderAgents, renderAgent)
		agentManager.renderAgents[name] = renderAgents
		agentManager.maxWork[name] = maxWorkIncrease
		agentManager.activeWork[name] = make([]string, 0, 0)
		return
	}

	renderAgents = append(renderAgents, renderAgent)
	agentManager.renderAgents[name] = renderAgents

	maxWork := agentManager.maxWork[name]
	agentManager.maxWork[name] = maxWork + maxWorkIncrease
}

func (agentManager *RenderAgentManager) run() {
	for {
		select {
		case ch, ok := <-agentManager.stop:
			{
				if !ok {
					return
				}
				ch <- true
				return
			}
		case statusUpdate, ok := <-agentManager.workStatus:
			{
				if !ok {
					return
				}
				log.Println("received status update", statusUpdate)
				agentManager.handleStatus(statusUpdate)
			}
		case <-time.After(5 * time.Second):
			{
				agentManager.dispatchMoreWork()
			}
		}
	}
}

func (agentManager *RenderAgentManager) dispatchMoreWork() {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()

	log.Println("About to look for work.")
	for name, renderAgents := range agentManager.renderAgents {
		log.Println("Looking for work for", name)
		workCount := agentManager.workToDispatchCount(name)
		rendererCount := len(renderAgents)
		log.Println("workCount", workCount, "rendererCount", rendererCount)
		if workCount > 0 && rendererCount > 0 {
			renderAgent := renderAgents[0]
			generatedAssets, err := agentManager.generatedAssetStorageManager.FindWorkForService(name, workCount)
			if err == nil {
				log.Println("Found", len(generatedAssets), "for", name)
				for _, generatedAsset := range generatedAssets {
					generatedAsset.Status = common.GeneratedAssetStatusScheduled
					err := agentManager.generatedAssetStorageManager.Update(generatedAsset)
					if err == nil {
						agentManager.activeWork[name] = uniqueListWith(agentManager.activeWork[name], generatedAsset.Id)
						renderAgent.Dispatch() <- generatedAsset.Id
					}
				}
			} else {
				log.Println("Error getting generated assets", err)
			}
		}
	}
}

func (agentManager *RenderAgentManager) handleStatus(renderStatus RenderStatus) {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()
	if renderStatus.Status == common.GeneratedAssetStatusComplete || strings.HasPrefix(renderStatus.Status, common.GeneratedAssetStatusFailed) {
		activeWork, hasActiveWork := agentManager.activeWork[renderStatus.Service]
		if hasActiveWork {
			agentManager.activeWork[renderStatus.Service] = listWithout(activeWork, renderStatus.GeneratedAssetId)
		}
	}
}

func (agentManager *RenderAgentManager) RemoveWork(service, id string) {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()
	activeWork, hasActiveWork := agentManager.activeWork[service]
	if hasActiveWork {
		agentManager.activeWork[service] = listWithout(activeWork, id)
	} else {
		log.Println("Warning: Called RemoveWork without any work to remove")
	}
}

func (agentManager *RenderAgentManager) workToDispatchCount(name string) int {
	activework, hasActiveWork := agentManager.activeWork[name]
	maxWork, hasMaxWork := agentManager.maxWork[name]
	if hasActiveWork && hasMaxWork {
		activeWorkCount := len(activework)
		if activeWorkCount < maxWork {
			return maxWork - activeWorkCount
		}
	}
	return 0
}

func listWithout(values []string, value string) []string {
	results := make([]string, 0, 0)
	for _, listValue := range values {
		if listValue != value {
			results = append(results, listValue)
		}
	}
	return results
}

func uniqueListWith(values []string, value string) []string {
	if values == nil {
		results := make([]string, 0, 1)
		results[0] = value
		return results
	}
	for _, ele := range values {
		if ele == value {
			return values
		}
	}
	return append(values, value)
}
