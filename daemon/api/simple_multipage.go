package api

import (
	"encoding/json"
	"github.com/ngerakines/preview/common"
	"log"
	"strings"
)

func (blueprint *simpleBlueprint) multipagePreviewInfoRequest(fileIds []string) ([]byte, error) {
	responseCollection := common.NewMultipageView()

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

func (blueprint *simpleBlueprint) composeMultipagePreviewView(fileId string, templates map[string]templateTuple) *common.MultipagePreviewView {
	sourceAsset, err := blueprint.getOriginSourceAsset(fileId)
	view := common.NewMultipagePreviewView()

	emptyPage := common.NewPageView()
	blueprint.fillMultipagePlaceholders(emptyPage, "unknown")

	view.SetPage(0, emptyPage)

	if err != nil {
		return view
	}

	generatedAssets, err := blueprint.generatedAssetStorageManager.FindBySourceAssetId(fileId)
	if err != nil {
		return view
	}

	fileType := blueprint.getSourceAssetType(sourceAsset)

	pagedGeneratedAssetSet := blueprint.groupGeneratedAssetsByPage(generatedAssets)
	for page, pagedGeneratedAssets := range pagedGeneratedAssetSet {
		pv := new(common.PageView)
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
		view.SetPage(page, pv)
	}

	return view
}

func (blueprint *simpleBlueprint) composePageInfoView(generatedAsset *common.GeneratedAsset, fileType, placeholderSize string, page int32) *common.PageInfoView {
	log.Println("Building preview image for", generatedAsset)
	if generatedAsset.Status == common.GeneratedAssetStatusComplete {
		signedUrl, expires := blueprint.signUrl(blueprint.scrubUrl(generatedAsset, placeholderSize))
		width, height, err := blueprint.getImageSize(generatedAsset)
		if err == nil {
			return common.NewPageInfoView(signedUrl, width, height, expires, "complete")
		}
	}
	if strings.HasPrefix(generatedAsset.Status, common.GeneratedAssetStatusFailed) {
		return blueprint.getMultipagePlaceholder(fileType, placeholderSize, generatedAsset.Status)
	}
	return blueprint.getMultipagePlaceholder(fileType, placeholderSize, "incomplete")
}

func (blueprint *simpleBlueprint) getMultipagePlaceholder(fileType, placeholderSize string, status string) *common.PageInfoView {
	placeholder := blueprint.placeholderManager.Url(fileType, placeholderSize)
	signedUrl, expires := blueprint.signUrl(blueprint.edgeContentHost + "/static" + placeholder.Url)
	return common.NewPageInfoView(signedUrl, int32(placeholder.Height), int32(placeholder.Width), expires, "incomplete")
}

func (blueprint *simpleBlueprint) fillMultipagePlaceholders(view *common.PageView, fileType string) {
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
