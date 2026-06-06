package agenthandler

import (
	"mifer/internal/domain"
	"sync"
)

type AgentHandler struct {
	agentService domain.AgentService
	mu           sync.RWMutex
}
