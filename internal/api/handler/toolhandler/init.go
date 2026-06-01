// Package toolhandler 提供工具确认相关的 HTTP API 处理器。
// 包括工具确认（confirm）和命令白名单管理。
package toolhandler

import (
	"mifer/internal/ai/confirm"
)

// ToolHandler 工具确认 API 处理器。
type ToolHandler struct {
	ConfirmStore *confirm.Store
	Workdir      string // 项目工作目录，用于 allowlist 文件路径
}

// NewToolHandler 创建工具确认处理器。
func NewToolHandler(store *confirm.Store, workdir string) *ToolHandler {
	return &ToolHandler{
		ConfirmStore: store,
		Workdir:      workdir,
	}
}
