// Package toolhandler 提供工具确认相关的 HTTP API 处理器。
// 包括工具确认（confirm）和命令白名单管理。
package toolhandler

import (
	"mifer/internal/domain"
	"sync"
)

// ToolHandler 工具确认 API 处理器。
type ToolHandler struct {
	toolService domain.ToolService
	mu          sync.RWMutex
}

// NewToolHandler 创建工具确认处理器。
func NewToolHandler(toolService domain.ToolService) *ToolHandler {
	return &ToolHandler{toolService: toolService}
}

// getService 安全获取当前服务实例
func (h *ToolHandler) getService() domain.ToolService {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.toolService
}

// SwapService 原子替换当前服务实例，返回旧服务
func (h *ToolHandler) SwapService(svc domain.ToolService) domain.ToolService {
	h.mu.Lock()
	defer h.mu.Unlock()
	old := h.toolService
	h.toolService = svc
	return old
}
