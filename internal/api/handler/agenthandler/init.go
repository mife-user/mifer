package agenthandler

import (
	"mifer/internal/domain"
	"sync"
)

type AgentHandler struct {
	agentService domain.AgentService
	mu           sync.RWMutex
}

func NewAgentHandler(agentService domain.AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

// getService 安全获取当前服务实例
func (h *AgentHandler) getService() domain.AgentService {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.agentService
}

// SwapService 原子替换当前服务实例，返回旧服务
func (h *AgentHandler) SwapService(svc domain.AgentService) domain.AgentService {
	h.mu.Lock()
	defer h.mu.Unlock()
	old := h.agentService
	h.agentService = svc
	return old
}
