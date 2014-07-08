package api

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"log"
	"strings"
)

func (blueprint *simpleBlueprint) multipagePreviewInfoRequest(fileIds []string) ([]byte, error) {
	responseCollection := make(map[string]*multipagePreviewView)

	templates, err := blueprint.legacyTemplates()
	if err != nil {
		return nil, err
	}

	for _, fileId := range fileIds {
		view := blueprint.composeMultipagePreviewView(fileId, templates)
		responseCollection[fileId] = view
	}

	return json.Marshal(responseCollection)
}

func (blueprint *simpleBlueprint) composeMultipagePreviewView(fileId string, templates map[string]templateTuple) *multipagePreviewView {
	sourceAsset, err := blueprint.getOriginSourceAsset(fileId)
	view := new(multipagePreviewView)
	view.Pages = make(map[string]*pageView)

	view.PageCount = 1
	emptyPage := new(pageView)
	blueprint.fillMultipagePlaceholders(emptyPage, "unknown")

	view.Pages["0"] = emptyPage

	if err != nil {
		return view
	}

	generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId(fileId)
	if err != nil {
		return view
	}

	fileType := blueprint.getSourceAssetType(sourceAsset)

	pagedGeneratedAssetSet := blueprint.groupGeneratedAssetsByPage(generatedAssets)
	view.PageCount = int32(len(pagedGeneratedAssetSet))
	for page, pagedGeneratedAssets := range pagedGeneratedAssetSet {
		pv := new(pageView)
		for _, generatedAsset := range pagedGeneratedAssets {
			templateTuple, hasTemplateTuple := templates[generatedAsset.TemplateId]
			if hasTemplateTuple {
				switch templateTuple.placeholderSize {
				case common.PlaceholderSizeSmall:
					pv.Small = blueprint.composePageInfoView(generatedAsset, fileType, templateTuple.placeholderSize, page)
				case common.PlaceholderSizeMedium:
					pv.Medium = blueprint.composePageInfoView(generatedAsset, fileType, templateTuple.placeholderSize, page)
				case common.PlaceholderSizeLarge:
					pv.Large = blueprint.composePageInfoView(generatedAsset, fileType, templateTuple.placeholderSize, page)
				case common.PlaceholderSizeJumbo:
					pv.Jumbo = blueprint.composePageInfoView(generatedAsset, fileType, templateTuple.placeholderSize, page)
				}
			}
		}
		blueprint.fillMultipagePlaceholders(pv, fileType)
		view.Pages[fmt.Sprintf("%d", page)] = pv
	}

	return view
}

func (blueprint *simpleBlueprint) composePageInfoView(generatedAsset *common.GeneratedAsset, fileType, placeholderSize string, page int32) *pageInfoView {
	log.Println("Building preview image for", generatedAsset)
	if generatedAsset.Status == common.GeneratedAssetStatusComplete {
		signedUrl, expires := blueprint.signUrl(blueprint.scrubUrl(generatedAsset, placeholderSize))
		width, height, err := blueprint.getImageSize(generatedAsset)
		if err == nil {
			return newPageInfoView(signedUrl, width, height, expires, "complete")
		}
	}
	if strings.HasPrefix(generatedAsset.Status, common.GeneratedAssetStatusFailed) {
		return blueprint.getMultipagePlaceholder(fileType, placeholderSize, generatedAsset.Status)
	}
	return blueprint.getMultipagePlaceholder(fileType, placeholderSize, "incomplete")
}

func (blueprint *simpleBlueprint) getMultipagePlaceholder(fileType, placeholderSize string, status string) *pageInfoView {
	placeholder := blueprint.placeholderManager.Url(fileType, placeholderSize)
	signedUrl, expires := blueprint.signUrl(blueprint.edgeContentHost + "/static" + placeholder.Url)
	return newPageInfoView(signedUrl, int32(placeholder.Height), int32(placeholder.Width), expires, "incomplete")
}

func (blueprint *simpleBlueprint) fillMultipagePlaceholders(view *pageView, fileType string) {
	if view.Small == nil {
		view.Small = blueprint.getMultipagePlaceholder(fileType, common.PlaceholderSizeSmall, "incomplete")
	}
	if view.Medium == nil {
		view.Medium = blueprint.getMultipagePlaceholder(fileType, common.PlaceholderSizeMedium, "incomplete")
	}
	if view.Large == nil {
		view.Large = blueprint.getMultipagePlaceholder(fileType, common.PlaceholderSizeLarge, "incomplete")
	}
	if view.Jumbo == nil {
		view.Jumbo = blueprint.getMultipagePlaceholder(fileType, common.PlaceholderSizeJumbo, "incomplete")
	}
}

func newPageInfoView(url string, height, width int32, expires int64, status string) *pageInfoView {
	return &pageInfoView{
		Url:     url,
		Width:   width,
		Height:  height,
		Expires: expires,
		Status:  status,
	}
}
