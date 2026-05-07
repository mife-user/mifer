package agenthandler

import "mifer/internal/domain"

type AgentHandler struct {
	AgentService domain.AgentService
}

func NewAgentHandler(agentService domain.AgentService) *AgentHandler {
	return &AgentHandler{AgentService: agentService}
}
