package agenthandler

import "mifer/internal/service/agentservice"

// CloseExecutor 释放底层 Executor 持有的资源（MCP 子进程、确认存储 actor 等）。
func (h *AgentHandler) CloseExecutor() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if svc, ok := h.agentService.(*agentservice.AgentService); ok {
		svc.CloseExecutor()
	}
}
