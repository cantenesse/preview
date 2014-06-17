package api

import (
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"log"
	"net/http"
)

type webHookBlueprint struct {
	base string
	gasm common.GeneratedAssetStorageManager
}

func NewWebHookBlueprint(gasm common.GeneratedAssetStorageManager) *webHookBlueprint {
	blueprint := new(webHookBlueprint)
	// TODO: Abstract this so WebHookBlueprint can apply to non-Zencoder web hooks as well
	blueprint.base = "/zencoder"
	blueprint.gasm = gasm
	return blueprint
}

func (blueprint *webHookBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Get(blueprint.buildUrl("/:id"), http.HandlerFunc(blueprint.zencoderApiHandler))
}

func (blueprint *webHookBlueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func (blueprint *webHookBlueprint) zencoderApiHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":id")
	ga, err := blueprint.gasm.FindById(id)
	if err != nil {
		log.Println("Could not find GeneratedAsset with ID", id, "in Zencoder web hook")
		return
	}
	ga.Status = common.GeneratedAssetStatusComplete
	log.Println("Transcoding complete for", id)
}
