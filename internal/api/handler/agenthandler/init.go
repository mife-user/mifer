package agenthandler

import "mifer/internal/domain"

type AgentHandler struct {
	agentService *domain.AgentService
}

func NewAgentHandler(agentService *domain.AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}
