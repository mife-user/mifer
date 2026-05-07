package agentservice

import (
	"mifer/internal/domain"
)

type AgentService struct {
	Executor domain.Agent
}

func NewAgentService(executor domain.Agent) domain.AgentService {
	return &AgentService{Executor: executor}
}
