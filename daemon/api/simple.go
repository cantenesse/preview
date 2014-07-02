package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/rcrowley/go-metrics"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type simpleBlueprint struct {
	base                         string
	edgeContentHost              string
	renderAgentManager           *agent.RenderAgentManager
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	placeholderManager           common.PlaceholderManager
	signatureManager             SignatureManager
	supportedFileTypes           map[string]int64
	generatePreviewRequestsMeter metrics.Meter
	previewInfoRequestsMeter     metrics.Meter
}

type templateTuple struct {
	placeholderSize string
	template        *common.Template
}

// NewSimpleBlueprint creates a new simpleBlueprint object.
func NewSimpleBlueprint(
	registry metrics.Registry,
	base string,
	edgeContentHost string,
	renderAgentManager *agent.RenderAgentManager,
	sourceAssetStorageManager common.SourceAssetStorageManager,
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	placeholderManager common.PlaceholderManager,
	signatureManager SignatureManager,
	supportedFileTypes map[string]int64) (*simpleBlueprint, error) {
	blueprint := new(simpleBlueprint)
	blueprint.base = base
	blueprint.edgeContentHost = edgeContentHost
	blueprint.renderAgentManager = renderAgentManager
	blueprint.sourceAssetStorageManager = sourceAssetStorageManager
	blueprint.generatedAssetStorageManager = generatedAssetStorageManager
	blueprint.templateManager = templateManager
	blueprint.placeholderManager = placeholderManager
	blueprint.supportedFileTypes = supportedFileTypes
	blueprint.signatureManager = signatureManager

	blueprint.generatePreviewRequestsMeter = metrics.NewMeter()
	blueprint.previewInfoRequestsMeter = metrics.NewMeter()
	registry.Register("simpleApi.generatePreviewRequests", blueprint.generatePreviewRequestsMeter)
	registry.Register("simpleApi.previewInfoRequests", blueprint.previewInfoRequestsMeter)

	return blueprint, nil
}

func (blueprint *simpleBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Put(blueprint.buildUrl("/v1/preview/"), http.HandlerFunc(blueprint.generatePreviewHandler))
	p.Put(blueprint.buildUrl("/v1/preview/:fileid"), http.HandlerFunc(blueprint.generatePreviewHandler))
	p.Get(blueprint.buildUrl("/v1/preview/"), http.HandlerFunc(blueprint.previewInfoHandler))
	p.Get(blueprint.buildUrl("/v1/preview/:fileid"), http.HandlerFunc(blueprint.previewInfoHandler))
	p.Get(blueprint.buildUrl("/v2/preview/"), http.HandlerFunc(blueprint.multipagePreviewInfoHandler))
	p.Get(blueprint.buildUrl("/v2/preview/:fileid"), http.HandlerFunc(blueprint.multipagePreviewInfoHandler))
}

func (blueprint *simpleBlueprint) generatePreviewHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.generatePreviewRequestsMeter.Mark(1)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(res, http.StatusText(400), 400)
		return
	}
	defer req.Body.Close()

	id, hasId := blueprint.urlHasFileId(req.URL.Path)
	if hasId {
		gprs, err := newGeneratePreviewRequestFromText(id, string(body))
		if err != nil {
			http.Error(res, http.StatusText(400), 400)
			return
		}
		blueprint.handleGeneratePreviewRequest(gprs)
	} else {
		gprs, err := newGeneratePreviewRequestFromJson(string(body))
		if err != nil {
			http.Error(res, http.StatusText(400), 400)
			return
		}
		blueprint.handleGeneratePreviewRequest(gprs)
	}

	res.Header().Set("Content-Length", "0")
	res.WriteHeader(202)
}

func (blueprint *simpleBlueprint) previewInfoHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.previewInfoRequestsMeter.Mark(1)

	fileIds := blueprint.parseFileIds(req)
	previewInfo, err := blueprint.handlePreviewInfoRequest(fileIds)
	if err != nil {
		http.Error(res, http.StatusText(500), 500)
		return
	}

	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(previewInfo))
}

func (blueprint *simpleBlueprint) multipagePreviewInfoHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.previewInfoRequestsMeter.Mark(1)

	fileIds := blueprint.parseFileIds(req)
	previewInfo, err := blueprint.multipagePreviewInfoRequest(fileIds)
	if err != nil {
		http.Error(res, http.StatusText(500), 500)
		return
	}

	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(previewInfo))
}

func (blueprint *simpleBlueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func (blueprint *simpleBlueprint) urlHasFileId(url string) (string, bool) {
	index := len(blueprint.buildUrl("/v1/preview/"))
	if len(url) > index {
		return url[index:], true
	}
	return "", false
}

func (blueprint *simpleBlueprint) handleGeneratePreviewRequest(gprs []*common.GeneratePreviewRequest) {
	for _, gpr := range gprs {
		blueprint.renderAgentManager.CreateWork(gpr.Id(), gpr.Url(), gpr.RequestType(), gpr.Size())
	}
}

func (blueprint *simpleBlueprint) parseFileIds(req *http.Request) []string {
	results := make([]string, 0, 0)

	// NKG: See if the url contains a file id
	url := req.URL.Path
	index := len(blueprint.buildUrl("/v1/preview/"))
	if len(url) > index {
		fileIds := strings.Split(url[index:], ",")
		for _, fileId := range fileIds {
			results = append(results, fileId)
		}
	}

	// NKG: Pull any file ids from the query string parameters.
	queryValues := req.URL.Query()
	for key, values := range queryValues {
		if key == "file_id" {
			for _, value := range values {
				fileIds := strings.Split(value, ",")
				for _, fileId := range fileIds {
					results = append(results, fileId)
				}
			}
		}
	}
	return results
}

func (blueprint *simpleBlueprint) getSourceAssetType(sourceAsset *common.SourceAsset) string {
	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		return fileType
	}
	return "unknown"
}

func (blueprint *simpleBlueprint) legacyTemplates() (map[string]templateTuple, error) {
	legacyTemplates, err := blueprint.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		return nil, err
	}

	templates := make(map[string]templateTuple)
	for _, legacyTemplate := range legacyTemplates {
		placeholderSize, err := blueprint.templatePlaceholderSize(legacyTemplate)
		if err != nil {
			return nil, err
		}
		templates[legacyTemplate.Id] = templateTuple{placeholderSize, legacyTemplate}
	}
	return templates, nil
}

func (blueprint *simpleBlueprint) handlePreviewInfoRequest(fileIds []string) ([]byte, error) {
	response := common.NewPreviewInfoResponse("v1")

	templates, err := blueprint.legacyTemplates()
	if err != nil {
		return nil, err
	}

	for _, fileId := range fileIds {
		sourceAsset, err := blueprint.getOriginSourceAsset(fileId)
		if err == nil {
			fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
			if err != nil {
				fileType = "unknown"
			}

			generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId(fileId)
			if err != nil {
				return nil, err
			}
			log.Println("generated assets for ", fileId, ":", generatedAssets)

			pagedGeneratedAssetSet := blueprint.groupGeneratedAssetsByPage(generatedAssets)
			for page, pagedGeneratedAssets := range pagedGeneratedAssetSet {
				collection := common.NewPreviewInfoCollection()
				collection.FileId = fileId

				for _, generatedAsset := range pagedGeneratedAssets {
					templateTuple, hasTemplateTuple := templates[generatedAsset.TemplateId]
					if hasTemplateTuple {
						switch templateTuple.placeholderSize {
						case common.PlaceholderSizeSmall:
							collection.Small = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize, page)
						case common.PlaceholderSizeMedium:
							collection.Medium = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize, page)
						case common.PlaceholderSizeLarge:
							collection.Large = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize, page)
						case common.PlaceholderSizeJumbo:
							collection.Jumbo = blueprint.getPreviewImage(generatedAsset, fileType, templateTuple.placeholderSize, page)
						}
					}
				}
				if collection.Small == nil {
					collection.Small = blueprint.getPlaceholder(fileType, common.PlaceholderSizeSmall, page)
				}
				if collection.Medium == nil {
					collection.Medium = blueprint.getPlaceholder(fileType, common.PlaceholderSizeMedium, page)
				}
				if collection.Large == nil {
					collection.Large = blueprint.getPlaceholder(fileType, common.PlaceholderSizeLarge, page)
				}
				if collection.Jumbo == nil {
					collection.Jumbo = blueprint.getPlaceholder(fileType, common.PlaceholderSizeJumbo, page)
				}
				response.AddCollection(collection)
			}
		} else {
			collection := common.NewPreviewInfoCollection()
			collection.FileId = fileId
			if collection.Small == nil {
				collection.Small = blueprint.getPlaceholder("unknown", common.PlaceholderSizeSmall, 0)
			}
			if collection.Medium == nil {
				collection.Medium = blueprint.getPlaceholder("unknown", common.PlaceholderSizeMedium, 0)
			}
			if collection.Large == nil {
				collection.Large = blueprint.getPlaceholder("unknown", common.PlaceholderSizeLarge, 0)
			}
			if collection.Jumbo == nil {
				collection.Jumbo = blueprint.getPlaceholder("unknown", common.PlaceholderSizeJumbo, 0)
			}

			response.AddCollection(collection)
		}
	}

	return json.Marshal(response)
}

func (blueprint *simpleBlueprint) groupGeneratedAssetsByPage(generatedAssets []*common.GeneratedAsset) map[int32][]*common.GeneratedAsset {
	results := make(map[int32][]*common.GeneratedAsset)
	for _, generatedAsset := range generatedAssets {
		page := blueprint.getGeneratedAssetPage(generatedAsset)
		generatedAssetsForPage, hasGeneratedAssetsForPage := results[page]
		if !hasGeneratedAssetsForPage {
			generatedAssetsForPage = make([]*common.GeneratedAsset, 0, 0)
		}
		generatedAssetsForPage = append(generatedAssetsForPage, generatedAsset)
		results[page] = generatedAssetsForPage
	}
	return results
}

func (blueprint *simpleBlueprint) getGeneratedAssetPage(generatedAsset *common.GeneratedAsset) int32 {
	var page int32 = 0
	pageValue, err := common.GetFirstAttribute(generatedAsset, common.GeneratedAssetAttributePage)
	if err == nil {
		parsedInt, err := strconv.ParseInt(pageValue, 10, 32)
		if err == nil {
			page = int32(parsedInt)
		}
	}
	return page
}

func (blueprint *simpleBlueprint) scrubUrl(generatedAsset *common.GeneratedAsset, placeholderSize string) string {
	page := blueprint.getGeneratedAssetPage(generatedAsset)
	return fmt.Sprintf("%s/asset/%s/%s/%d", blueprint.edgeContentHost, generatedAsset.SourceAssetId, placeholderSize, page)
}

func (blueprint *simpleBlueprint) signUrl(url string) (string, int64) {
	// NKG: Configuration should be added to determine if urls should be signed or not.
	signedUrl, expires, err := blueprint.signatureManager.Sign(url)
	if err != nil {
		return url, 0
	}
	return signedUrl, expires
}

func (blueprint *simpleBlueprint) templatePlaceholderSize(template *common.Template) (string, error) {
	if !template.HasAttribute(common.TemplateAttributePlaceholderSize) {
		// TODO: write this code
		return "", common.ErrorNotImplemented
	}
	placeholderSizes := template.GetAttribute(common.TemplateAttributePlaceholderSize)
	if len(placeholderSizes) < 1 {
		// TODO: write this code
		return "", common.ErrorNotImplemented
	}
	placeholderSize := placeholderSizes[0]
	return placeholderSize, nil
}

func (blueprint *simpleBlueprint) getPreviewImage(generatedAsset *common.GeneratedAsset, fileType, placeholderSize string, page int32) *common.ImageInfo {
	log.Println("Building preview image for", generatedAsset)
	if generatedAsset.Status == common.GeneratedAssetStatusComplete {
		signedUrl, expires := blueprint.signUrl(blueprint.scrubUrl(generatedAsset, placeholderSize))
		width, height, err := blueprint.getImageSize(generatedAsset)
		if err != nil {
			return common.NewImageInfo(signedUrl, 200, 200, expires, true, false)
		}
		return common.NewImageInfo(signedUrl, width, height, expires, true, false)
	}
	if strings.HasPrefix(generatedAsset.Status, common.GeneratedAssetStatusFailed) {
		// NKG: If the job failed, then before we return the placeholder, we set the "isFinal" field.
		placeholder := blueprint.getPlaceholder(fileType, placeholderSize, page)
		placeholder.IsFinal = true
		return placeholder
	}
	return blueprint.getPlaceholder(fileType, placeholderSize, page)
}

func (blueprint *simpleBlueprint) getImageSize(ga *common.GeneratedAsset) (int32, int32, error) {
	if !(ga.HasAttribute("imageWidth") && ga.HasAttribute("imageHeight")) {
		log.Println("Asset does not have width and height attributes")
		return 0, 0, common.ErrorNotImplemented
	}
	widths := ga.GetAttribute("imageWidth")
	heights := ga.GetAttribute("imageHeight")
	if len(widths) == 0 || len(heights) == 0 {
		log.Println("Asset does not have width and height attributes")
		return 0, 0, common.ErrorNotImplemented
	}
	width, err := strconv.Atoi(widths[0])
	if err != nil {
		log.Println("Error parsing template width")
		return 0, 0, err
	}
	height, err := strconv.Atoi(heights[0])
	if err != nil {
		log.Println("Error parsing template height")
		return 0, 0, err
	}
	return int32(width), int32(height), nil
}

func (blueprint *simpleBlueprint) getPlaceholder(fileType, placeholderSize string, page int32) *common.ImageInfo {
	placeholder := blueprint.placeholderManager.Url(fileType, placeholderSize)
	signedUrl, expires := blueprint.signUrl(blueprint.edgeContentHost + "/static" + placeholder.Url)
	return common.NewImageInfo(signedUrl, 200, 200, expires, false, true)
}

func (blueprint *simpleBlueprint) getFileType(sourceAssets []*common.SourceAsset) string {
	if len(sourceAssets) > 0 {
		sourceAsset := sourceAssets[0]
		if sourceAsset.HasAttribute(common.SourceAssetAttributeType) {
			fileTypes := sourceAsset.GetAttribute(common.SourceAssetAttributeType)
			if len(fileTypes) > 0 {
				return fileTypes[0]
			}
		}
	}
	return common.DefaultPlaceholderType
}

func (blueprint *simpleBlueprint) getOriginSourceAsset(generatedAssetId string) (*common.SourceAsset, error) {
	sourceAssets, err := blueprint.sourceAssetStorageManager.FindBySourceAssetId(generatedAssetId)
	if err != nil {
		return nil, err
	}
	for _, sourceAsset := range sourceAssets {
		if sourceAsset.IdType == common.SourceAssetTypeOrigin {
			return sourceAsset, nil
		}
	}
	return nil, common.ErrorNoSourceAssetsFoundForId
}
