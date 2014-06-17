package api

import(
	"log"
	"net/http"
	"strconv"
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/render"
	"github.com/ngerakines/preview/common"
	"io/ioutil"
	"encoding/json"
)

type apiV2Blueprint struct {
	base string
	agentManager *render.RenderAgentManager
	gasm common.GeneratedAssetStorageManager
	sasm common.SourceAssetStorageManager
}

type generatePreviewRequestV2 struct {
	id	    string
	url	    string
	attributes  map[string][]string
	templateIds  []string
}

func NewApiV2Blueprint(
	base string,
	agentManager *render.RenderAgentManager,
	gasm common.GeneratedAssetStorageManager,
	sasm common.SourceAssetStorageManager) *apiV2Blueprint{
	bp := new(apiV2Blueprint)
	bp.base = base
	bp.agentManager = agentManager
	bp.gasm = gasm
	bp.sasm = sasm
	return bp
}

func (blueprint *apiV2Blueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Put(blueprint.buildUrl("/v2/preview/"), http.HandlerFunc(blueprint.GeneratePreviewHandler))

	p.Get(blueprint.buildUrl("/v2/preview/"), http.HandlerFunc(blueprint.PreviewQueryHandler)) // Search - /preview/?id=1234?id=5678
	p.Get(blueprint.buildUrl("/v2/preview/:id"), http.HandlerFunc(blueprint.PreviewInfoHandler)) // Get specific asset - /preview/12345
	p.Get(blueprint.buildUrl("/v2/preview/:id/:attribute"), http.HandlerFunc(blueprint.PreviewAttributeHandler)) // Specific attribute - /preview/123/data
}

func (blueprint *apiV2Blueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func (blueprint *apiV2Blueprint) PreviewInfoHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":id")
	jsonData, err := blueprint.marshalSourceAssetsFromIds([]string{id})
	if err != nil {
		log.Println("Marshalling error", err)
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}
	res.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	res.Write(jsonData)
}
func (blueprint *apiV2Blueprint) PreviewAttributeHandler(res http.ResponseWriter, req *http.Request) {

}

func (blueprint *apiV2Blueprint) GeneratePreviewHandler(res http.ResponseWriter, req *http.Request) {
	//TODO: metrics
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(400)
		return
	}
	defer req.Body.Close()
	gprs, err := newGeneratePreviewRequestV2(string(body))
	if err != nil {
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(400)
		return
	}
	blueprint.handleGeneratePreviewRequest(gprs)
	target := blueprint.buildUrl("/v2/preview/")
	for _, gpr := range gprs {
		target += "?id=" + gpr.id
	}
	http.Redirect(res, req, target, 303)
}

type sourceAssetView struct {
	SourceAssets []extendedSourcedAsset `json:"sourceAssets"`
}

type extendedSourcedAsset struct {
	SourceAsset *common.SourceAsset `json:"sourceAsset"`
	GeneratedAssets []*common.GeneratedAsset `json:"generatedAssets"`
}

type generatedAssetView struct {
	GeneratedAssetId string `json:"generatedAssetId"`
	Location string `json:"location"`
	Attributes []common.Attribute `json:"attributes"`
}

func (blueprint *apiV2Blueprint) PreviewQueryHandler(res http.ResponseWriter, req *http.Request) {
	ids, hasIds := req.URL.Query()["id"]
	if !hasIds {
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(400)
		return
	}

	jsonData, err := blueprint.marshalSourceAssetsFromIds(ids)
	if err != nil {
		log.Println("Marshalling error", err)
		res.Header().Set("Content-Length", "0")
		res.WriteHeader(500)
		return
	}
	
	res.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	res.Write(jsonData)
}

func (blueprint *apiV2Blueprint) marshalSourceAssetsFromIds(ids []string) ([]byte, error) {
	var data sourceAssetView
	
	for _, id := range ids {
		gas, err := blueprint.gasm.FindBySourceAssetId(id)
		if err != nil {
			return nil, err
		}
		// gaa := make([]generatedAssetView, 0)
		// for _, ga := range gas {
		// 	gaa = append(gaa, generatedAssetView{
		// 		GeneratedAssetId: ga.Id,
		// 		Location: ga.Location,
		// 		Attributes: ga.Attributes,
		// 	})
		// }
		sas, _ := blueprint.sasm.FindBySourceAssetId(id)
		for _, sa := range sas {
			data.SourceAssets = append(data.SourceAssets, extendedSourcedAsset{
				SourceAsset: sa,
				GeneratedAssets: gas,
			})
		}	
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

func newGeneratePreviewRequestV2(body string) ([]*generatePreviewRequestV2, error) {
	var data struct {
		SourceAssets   []struct {
			Id	    string `json:"fileId"`
			Url	    string `json:"url"`
			Attributes  map[string][]string `json:"attributes"`
		} `json:"sourceAssets"`
		TemplateIds []string `json:"templateIds"`
	}
	err := json.Unmarshal([]byte(body), &data)
	if err != nil {
		return nil, err
	}
	gprs := make([]*generatePreviewRequestV2, 0, 0)
	for _, sourceAsset := range(data.SourceAssets) {
		gpr := new(generatePreviewRequestV2)
		gpr.id = sourceAsset.Id
		gpr.url = sourceAsset.Url
		gpr.attributes = sourceAsset.Attributes
		gpr.templateIds = data.TemplateIds
		gprs = append(gprs, gpr)
	}
	log.Println("GPRS:", gprs)
	return gprs, nil
}

func (blueprint *apiV2Blueprint) handleGeneratePreviewRequest(gprs []*generatePreviewRequestV2) {
	for _, gpr := range gprs {
		blueprint.agentManager.CreateWorkFromTemplates(gpr.id, gpr.url, gpr.attributes, gpr.templateIds)
	}
}
