package api

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"log"
	"path/filepath"
	"strings"
)

var assetActionVideoURL = assetAction(5)

func (blueprint *apiV2Blueprint) getAsset(fileId, templateId, page string) (assetAction, string) {
	generatedAssets, err := blueprint.gasm.FindBySourceAssetId(fileId)
	if err != nil {
		log.Println("Error finding generated asset")
		return assetAction404, ""
	}
	if len(generatedAssets) == 0 {
		log.Println("Error finding generated asset")
		return assetAction404, ""
	}

	var generatedAsset *common.GeneratedAsset
	for _, ga := range generatedAssets {
		pageVal, _ := common.GetFirstAttribute(ga, common.GeneratedAssetAttributePage)
		if len(pageVal) == 0 {
			pageVal = "0"
		}
		if pageVal == page {
			generatedAsset = ga
			break
		}
	}
	surl := generatedAsset.GetAttribute("streamingUrl")
	if len(surl) > 0 && len(surl[0]) > 0 {
		return assetActionVideoURL, surl[0]
	}

	if strings.HasPrefix(generatedAsset.Location, "local://") {

		fullPath := filepath.Join(blueprint.localAssetStoragePath, generatedAsset.Location[8:])
		if util.CanLoadFile(fullPath) {
			return assetActionServeFile, fullPath
		} else {
			return assetAction404, ""
		}
	}

	if strings.HasPrefix(generatedAsset.Location, "s3://") {
		return assetActionS3Proxy, generatedAsset.Location
	}

	return assetAction404, ""
}
