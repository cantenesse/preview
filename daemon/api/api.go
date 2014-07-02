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
	"path/filepath"
	"net/url"
	"time"
)

type apiBlueprint struct {
	base                         string
	agentManager                 *agent.RenderAgentManager
	gasm                         common.GeneratedAssetStorageManager
	sasm                         common.SourceAssetStorageManager
	s3Client                     common.S3Client
	generatePreviewRequestsMeter metrics.Meter
	previewQueriesMeter          metrics.Meter
	previewInfoRequestsMeter     metrics.Meter
	previewGADataRequestsMeter   metrics.Meter
	previewGAInfoRequestsMeter   metrics.Meter
}

type apiGeneratePreviewRequest struct {
	id          string
	url         string
	attributes  map[string][]string
	templateIds []string
}

type userPreviewRequest struct {
	SourceAssets []struct {
		Id         string              `json:"fileId"`
		Url        string              `json:"url"`
		Attributes map[string][]string `json:"attributes"`
	} `json:"sourceAssets"`
	TemplateIds []string `json:"templateIds"`
}

type sourceAssetView struct {
	SourceAssets []extendedSourcedAsset `json:"sourceAssets"`
}

type extendedSourcedAsset struct {
	SourceAsset     *common.SourceAsset      `json:"sourceAsset"`
	GeneratedAssets []*common.GeneratedAsset `json:"generatedAssets"`
}

type GeneratedAssetList []*common.GeneratedAsset

func NewApiBlueprint(
	base string,
	agentManager *agent.RenderAgentManager,
	gasm common.GeneratedAssetStorageManager,
	sasm common.SourceAssetStorageManager,
	registry metrics.Registry,
	s3Client common.S3Client) *apiBlueprint {
	bp := new(apiBlueprint)
	bp.base = base
	bp.agentManager = agentManager
	bp.gasm = gasm
	bp.sasm = sasm
	bp.s3Client = s3Client

	bp.generatePreviewRequestsMeter = metrics.NewMeter()
	bp.previewQueriesMeter = metrics.NewMeter()
	bp.previewInfoRequestsMeter = metrics.NewMeter()
	bp.previewGADataRequestsMeter = metrics.NewMeter()
	bp.previewGAInfoRequestsMeter = metrics.NewMeter()

	registry.Register("api.generatePreviewRequests", bp.generatePreviewRequestsMeter)
	registry.Register("api.previewQueries", bp.previewQueriesMeter)
	registry.Register("api.previewInfoRequests", bp.previewInfoRequestsMeter)
	registry.Register("api.previewGADataRequests", bp.previewGADataRequestsMeter)
	registry.Register("api.previewGAInfoRequests", bp.previewGAInfoRequestsMeter)

	return bp
}

func (blueprint *apiBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Put(blueprint.buildUrl("/preview/"), http.HandlerFunc(blueprint.generatePreviewHandler))
	p.Get(blueprint.buildUrl("/preview/:id/:templateid/:page/data"), http.HandlerFunc(blueprint.previewGADataHandler))
	p.Get(blueprint.buildUrl("/preview/:id/:templateid/:page"), http.HandlerFunc(blueprint.previewGAInfoHandler))
	p.Get(blueprint.buildUrl("/preview/:id/:templateid"), http.HandlerFunc(blueprint.previewGAInfoHandler)) // Generated assets with template ID - /preview/123/456
	p.Get(blueprint.buildUrl("/preview/:id"), http.HandlerFunc(blueprint.previewInfoHandler))               // Get specific source assets with ID - /preview/12345
	p.Get(blueprint.buildUrl("/preview/"), http.HandlerFunc(blueprint.previewQueryHandler))                 // Search - /preview/?id=1234&id=5678
}

func (blueprint *apiBlueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func getUrl(id string) string {
	return "file:///Users/james.bloxham/Documents/LittleBitOfEverything.docx"
}

func (blueprint *apiBlueprint) previewQueryHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.previewQueriesMeter.Mark(1)

	ids, hasIds := req.URL.Query()["id"]
	if !hasIds {
		http.Error(res, "", 400)
		return
	}

	if true {
		log.Println("1")
		for _, id := range ids {
			log.Println("2")
			sas, _ := blueprint.sasm.FindBySourceAssetId(id)
			if len(sas) == 0 {
				url := getUrl(id)
				attributes := make(map[string][]string)
				attributes["type"] = []string{filepath.Ext(url[5:])[1:]}
				templateIds := []string{common.DocumentConversionTemplateId}
				blueprint.agentManager.CreateWorkFromTemplates(id, url, attributes, templateIds)
			}
		}

		time.Sleep(15 * time.Second)
	}

	jsonData, err := blueprint.marshalSourceAssetsFromIds(ids)
	if err != nil {
		log.Println(err)
		http.Error(res, "", 500)
		return
	}

	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(jsonData))
}

func (blueprint *apiBlueprint) previewInfoHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.previewInfoRequestsMeter.Mark(1)
	id := req.URL.Query().Get(":id")

	jsonData, err := blueprint.marshalSourceAssetsFromIds([]string{id})
	if err != nil {
		log.Println(err)
		http.Error(res, "", 500)
		return
	}

	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(jsonData))
}

func (blueprint *apiBlueprint) previewGAInfoHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.previewGAInfoRequestsMeter.Mark(1)

	id := req.URL.Query().Get(":id")
	templateId := req.URL.Query().Get(":templateid")
	page := req.URL.Query().Get(":page")

	jsonData, err := blueprint.marshalGeneratedAssets(id, templateId, page)
	if err != nil {
		log.Println(err)
		http.Error(res, "", 500)
		return
	}

	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(jsonData))
}

func (blueprint *apiBlueprint) previewGADataHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.previewGADataRequestsMeter.Mark(1)

	id := req.URL.Query().Get(":id")
	templateId := req.URL.Query().Get(":templateid")
	page := req.URL.Query().Get(":page")

	http.Redirect(res, req, fmt.Sprintf("/asset/%s/%s/%s", id, templateId, page), 303)
}

func (blueprint *apiBlueprint) generatePreviewHandler(res http.ResponseWriter, req *http.Request) {
	blueprint.generatePreviewRequestsMeter.Mark(1)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "", 400)
		return
	}
	defer req.Body.Close()

	gprs, err := newApiGeneratePreviewRequest(string(body))
	if err != nil {
		http.Error(res, "", 400)
		return
	}

	for _, gpr := range gprs {
		blueprint.agentManager.CreateWorkFromTemplates(gpr.id, gpr.url, gpr.attributes, gpr.templateIds)
	}

	target := blueprint.buildUrl("/preview/?")
	params := url.Values{}
	for _, gpr := range gprs {
		params.Add("id", gpr.id)
	}
	target += params.Encode()

	http.Redirect(res, req, target, 303)
}

func (view *sourceAssetView) Serialize() ([]byte, error) {
	bytes, err := json.Marshal(view)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (blueprint *apiBlueprint) marshalSourceAssetsFromIds(ids []string) ([]byte, error) {
	var data sourceAssetView

	for _, id := range ids {
		gas, err := blueprint.gasm.FindBySourceAssetId(id)
		if err != nil {
			return nil, err
		}
		sas, _ := blueprint.sasm.FindBySourceAssetId(id)
		for _, sa := range sas {
			uniqgas := make([]*common.GeneratedAsset, 0, len(gas))
			for _, ga := range gas {
				if ga.SourceAssetType == sa.IdType {
					uniqgas = append(uniqgas, ga)
				}
			}
			data.SourceAssets = append(data.SourceAssets, extendedSourcedAsset{
				SourceAsset:     sa,
				GeneratedAssets: uniqgas,
			})
		}
	}

	jsonData, err := data.Serialize()
	if err != nil {
		log.Println("Serialization error:", err)
		return nil, common.ErrorCouldNotSerializeSourceAssets
	}

	return jsonData, nil
}

func (gal GeneratedAssetList) Serialize(single bool) ([]byte, error) {
	var bytes []byte
	var err error
	if single {
		bytes, err = json.Marshal(gal[0])
	} else {
		bytes, err = json.Marshal(gal)
	}
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (blueprint *apiBlueprint) marshalGeneratedAssets(said, templateId, page string) ([]byte, error) {
	gas, err := blueprint.gasm.FindBySourceAssetId(said)
	if err != nil {
		return nil, err
	}

	var arr GeneratedAssetList
	for _, g := range gas {
		if g.TemplateId == templateId {
			if len(page) > 0 {
				gPage, err := common.GetFirstAttribute(g, common.GeneratedAssetAttributePage)
				if err != nil {
					gPage = "0"
				}
				if page == gPage {
					arr = append(arr, g)
					break
				}
			} else {
				arr = append(arr, g)
			}
		}
	}

	if len(arr) == 0 {
		log.Println("Could not find GeneratedAssets with source and template id", err)
		return nil, common.ErrorUnableToFindGeneratedAssetsById
	}

	// If the caller gave a page, return the asset itself. Otherwise return an array of GAs
	jsonData, err := arr.Serialize(len(page) > 0)
	if err != nil {
		log.Println("Serialization error:", err)
		return nil, common.ErrorCouldNotSerializeGeneratedAssets
	}
	return jsonData, nil
}

func newApiGeneratePreviewRequest(body string) ([]*apiGeneratePreviewRequest, error) {
	var data userPreviewRequest
	err := json.Unmarshal([]byte(body), &data)
	if err != nil {
		return nil, err
	}
	gprs := make([]*apiGeneratePreviewRequest, 0, 0)
	for _, sourceAsset := range data.SourceAssets {
		gpr := new(apiGeneratePreviewRequest)
		gpr.id = sourceAsset.Id
		gpr.url = sourceAsset.Url
		gpr.attributes = sourceAsset.Attributes
		gpr.templateIds = data.TemplateIds
		gprs = append(gprs, gpr)
	}
	return gprs, nil
}
