package api

import(
	"github.com/bmizerany/pat"
	"net/http"
	"log"
)

type webHookBlueprint struct {
	base string
	
}

func (blueprint * webHookBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Get(blueprint.buildUrl("/zencoder/:id"), http.HandlerFunc(blueprint.zencoderApiHandler))
}

func (blueprint *webHookBlueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func (blueprint *webHookBlueprint) zencoderApiHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":id")
	log.Println(id)
}
