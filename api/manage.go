package api

import (
	"bytes"
	"encoding/json"
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"net/http"
	"time"
)

type manageBlueprint struct {
	base string

	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
}

type expandedSourceAsset struct {
}

type sourceAssetsView struct {
	SourceAssets []*common.SourceAsset
}

type generatedAssetsView struct {
	GeneratedAssets []*common.GeneratedAsset
}

type idPair struct {
	Id     string
	IdType string
}

type deleteSourceAssetsView struct {
	SourceAssets []idPair
}

type deleteGeneratedAssetsView struct {
	Id string
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
	// NKG: This should return a collection of source asset records.
	id := req.URL.Query().Get(":id")
	// TODO[NKG]: Error checking
	sourceAssets, _ := blueprint.sourceAssetStorageManager.FindBySourceAssetId(id)
	if len(sourceAssets) > 0 {
		view := &sourceAssetsView{sourceAssets}
		data, _ := view.Serialize()
		// TODO[NKG]: Error checking
		http.ServeContent(res, req, "", time.Now(), bytes.NewReader(data))
		return
	}
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) getGeneratedAssetHandler(res http.ResponseWriter, req *http.Request) {
	// NKG: This should return a collection of source asset records.
	id := req.URL.Query().Get(":id")
	// TODO[NKG]: Error checking
	generatedAsset, _ := blueprint.generatedAssetStorageManager.FindById(id)
	if generatedAsset != nil {
		data, _ := generatedAsset.Serialize()
		// TODO[NKG]: Error checking
		http.ServeContent(res, req, "", time.Now(), bytes.NewReader(data))
		return
	}
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) deleteSourceAssetHandler(res http.ResponseWriter, req *http.Request) {
	// NKG: This should return a collection of source asset records.
	id := req.URL.Query().Get(":id")
	// TODO[NKG]: Error checking
	sourceAssets, _ := blueprint.sourceAssetStorageManager.FindBySourceAssetId(id)
	view := new(deleteSourceAssetsView)
	view.SourceAssets = make([]idPair, 0, 0)
	if len(sourceAssets) > 0 {
		for _, sourceAsset := range sourceAssets {
			err := blueprint.sourceAssetStorageManager.Delete(id, sourceAsset.IdType)
			if err == nil {
				view.SourceAssets = append(view.SourceAssets, idPair{id, sourceAsset.IdType})
			}
		}

	}
	data, _ := view.Serialize()
	// TODO[NKG]: Error checking
	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(data))
}

func (blueprint *manageBlueprint) deleteGeneratedAssetHandler(res http.ResponseWriter, req *http.Request) {
	// NKG: This should return a collection of source asset records.
	id := req.URL.Query().Get(":id")
	// TODO[NKG]: Error checking
	generatedAsset, _ := blueprint.generatedAssetStorageManager.FindById(id)
	if generatedAsset != nil {
		blueprint.generatedAssetStorageManager.Delete(id)
		view := &deleteGeneratedAssetsView{id}
		data, _ := view.Serialize()
		http.ServeContent(res, req, "", time.Now(), bytes.NewReader(data))
		return
	}
	http.NotFound(res, req)
}

func (blueprint *manageBlueprint) createActiveWork(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (view *sourceAssetsView) Serialize() ([]byte, error) {
	bytes, err := json.Marshal(view)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (view *deleteSourceAssetsView) Serialize() ([]byte, error) {
	bytes, err := json.Marshal(view)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (view *deleteGeneratedAssetsView) Serialize() ([]byte, error) {
	bytes, err := json.Marshal(view)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
