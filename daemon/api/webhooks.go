package api

import (
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"log"
	"net/http"
)

type webhookBlueprint struct {
	base               string
	gasm               common.GeneratedAssetStorageManager
	renderAgentManager *agent.RenderAgentManager
}

func NewWebhookBlueprint(gasm common.GeneratedAssetStorageManager, ram *agent.RenderAgentManager) *webhookBlueprint {
	blueprint := new(webhookBlueprint)
	// TODO: Abstract this so WebHookBlueprint can apply to non-Zencoder web hooks as well
	blueprint.base = "/webhooks"
	blueprint.gasm = gasm
	blueprint.renderAgentManager = ram
	return blueprint
}

func (blueprint *webhookBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Post(blueprint.buildUrl("/zencoder/:id"), http.HandlerFunc(blueprint.zencoderApiHandler))
}

func (blueprint *webhookBlueprint) buildUrl(path string) string {
	return blueprint.base + path
}

func (blueprint *webhookBlueprint) zencoderApiHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":id")
	ga, err := blueprint.gasm.FindById(id)
	if err != nil {
		log.Println("Could not find GeneratedAsset with ID", id, "in Zencoder web hook")
		http.Error(res, "", 500)
		return
	}
	ga.Status = common.GeneratedAssetStatusComplete
	log.Println("Updating", ga)
	blueprint.gasm.Update(ga)
	blueprint.renderAgentManager.RemoveWork("videoRenderAgent", id)
	log.Println("Transcoding complete for", id)

	res.Header().Set("Content-Length", "0")
	res.WriteHeader(202)
}
