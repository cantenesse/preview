package api

import (
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"net/http"
)

type manageBlueprint struct {
	base string

	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
}

func NewManageBlueprint(sourceAssetStorageManager common.SourceAssetStorageManager, generatedAssetStorageManager common.GeneratedAssetStorageManager) Blueprint {
	blueprint := new(manageBlueprint)
	blueprint.base = "/manage"
	blueprint.sourceAssetStorageManager = sourceAssetStorageManager
	blueprint.generatedAssetStorageManager = generatedAssetStorageManager
	return blueprint
}

func (blueprint *manageBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Get(blueprint.base+"/sourceAsset/:id", http.HandlerFunc(blueprint.getSourceAssetHandler))
	p.Get(blueprint.base+"/generatedAsset/:id", http.HandlerFunc(blueprint.getGeneratedAssetHandler))
	p.Del(blueprint.base+"/sourceAsset/:id", http.HandlerFunc(blueprint.deleteSourceAssetHandler))
	p.Del(blueprint.base+"/generatedAsset/:id", http.HandlerFunc(blueprint.deleteGeneratedAssetHandler))
	p.Put(blueprint.base+"/activeWork", http.HandlerFunc(blueprint.createActiveWork))
}

func (blueprint *manageBlueprint) getSourceAssetHandler(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) getGeneratedAssetHandler(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) deleteSourceAssetHandler(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) deleteGeneratedAssetHandler(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) createActiveWork(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}
