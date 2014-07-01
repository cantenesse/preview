package api

import (
	"bytes"
	"encoding/json"
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/rcrowley/go-metrics"
	"net/http"
	"strconv"
)

type adminBlueprint struct {
	base                 string
	registry             metrics.Registry
	config               string
	renderAgents         []string
	placeholderManager   common.PlaceholderManager
	temporaryFileManager common.TemporaryFileManager
	agentManager         *agent.RenderAgentManager
}

type placeholdersView struct {
	Placeholders []*common.Placeholder
}

type temporaryFilesView struct {
	Files map[string]int
}

type renderAgentViewElement struct {
	Count      int      `json:"count"`
	Enabled    bool     `json:"enabled"`
	ActiveWork []string `json:"activeWork"`
}

type renderAgentsView struct {
	RenderAgents map[string]renderAgentViewElement `json:"renderAgents"`
}

type errorViewError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type errorsView struct {
	Errors []errorViewError `json:"errors"`
}

// NewAdminBlueprint creates a new adminBlueprint object.
func NewAdminBlueprint(registry metrics.Registry,
	config string,
	renderAgents []string,
	placeholderManager common.PlaceholderManager,
	temporaryFileManager common.TemporaryFileManager,
	agentManager *agent.RenderAgentManager) *adminBlueprint {
	blueprint := new(adminBlueprint)
	blueprint.base = "/admin"
	blueprint.registry = registry
	blueprint.config = config
	blueprint.renderAgents = renderAgents
	blueprint.placeholderManager = placeholderManager
	blueprint.temporaryFileManager = temporaryFileManager
	blueprint.agentManager = agentManager
	return blueprint
}

func (blueprint *adminBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Get(blueprint.base+"/config", http.HandlerFunc(blueprint.configHandler))
	p.Get(blueprint.base+"/placeholders", http.HandlerFunc(blueprint.placeholdersHandler))
	p.Get(blueprint.base+"/temporaryFiles", http.HandlerFunc(blueprint.temporaryFilesHandler))
	p.Get(blueprint.base+"/errors", http.HandlerFunc(blueprint.errorsHandler))
	p.Get(blueprint.base+"/renderAgents", http.HandlerFunc(blueprint.renderAgentsHandler))
	p.Get(blueprint.base+"/metrics", http.HandlerFunc(blueprint.metricsHandler))
}

func (blueprint *adminBlueprint) configHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Length", strconv.Itoa(len(blueprint.config)))
	res.Write([]byte(blueprint.config))
}

func (blueprint *adminBlueprint) metricsHandler(res http.ResponseWriter, req *http.Request) {
	content := &bytes.Buffer{}
	enc := json.NewEncoder(content)
	enc.Encode(blueprint.registry)
	res.Header().Set("Content-Length", strconv.Itoa(content.Len()))
	res.Write(content.Bytes())
}

func (blueprint *adminBlueprint) placeholdersHandler(res http.ResponseWriter, req *http.Request) {
	view := new(placeholdersView)
	view.Placeholders = make([]*common.Placeholder, 0, 0)
	for _, fileType := range blueprint.placeholderManager.AllFileTypes() {
		for _, placeholderSize := range common.DefaultPlaceholderSizes {
			view.Placeholders = append(view.Placeholders, blueprint.placeholderManager.Url(fileType, placeholderSize))
		}
	}

	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}

func (blueprint *adminBlueprint) renderAgentsHandler(res http.ResponseWriter, req *http.Request) {
	view := new(renderAgentsView)
	view.RenderAgents = make(map[string]renderAgentViewElement)
	for _, name := range blueprint.renderAgents {
		view.RenderAgents[name] = blueprint.newRenderAgentViewElement(name)
	}

	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}

func (blueprint *adminBlueprint) newRenderAgentViewElement(name string) renderAgentViewElement {
	enabled, count, activeWork := blueprint.agentManager.ActiveWorkForRenderAgent(name)
	return renderAgentViewElement{count, enabled, activeWork}
}

func (blueprint *adminBlueprint) errorsHandler(res http.ResponseWriter, req *http.Request) {
	view := new(errorsView)
	view.Errors = make([]errorViewError, 0, 0)
	for _, err := range common.AllErrors {
		view.Errors = append(view.Errors, errorViewError{err.Error(), err.Description()})
	}
	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}

func (blueprint *adminBlueprint) temporaryFilesHandler(res http.ResponseWriter, req *http.Request) {
	view := new(temporaryFilesView)
	view.Files = blueprint.temporaryFileManager.List()

	body, err := json.Marshal(view)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Length", strconv.Itoa(len(body)))
	res.Write(body)
}
