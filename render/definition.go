package render

import "github.com/ngerakines/preview/common"

type RenderAgent interface {
	Stop()
	AddStatusListener(listener RenderStatusChannel)
	Dispatch() RenderAgentWorkChannel
}

type RenderAgentWorkChannel chan string

type RenderStatusChannel chan RenderStatus

type RenderStatus struct {
	GeneratedAssetId string
	Status           string
	Service          string
}

type generatedAssetUpdate struct {
	status     string
	attributes []common.Attribute
}
