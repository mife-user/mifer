package toolservice

import (
	"mifer/internal/ai/confirm"
	"mifer/internal/domain"
)

// ToolService 工具操作服务实现，实现 domain.ToolService 接口。
// 对 confirm.Store 和配置的操作在此层完成，Handler 不直接访问它们。
type ToolService struct {
	store   *confirm.Store
	workdir string // 项目工作目录，用于 allowlist 文件路径
}

// NewToolService 创建工具操作服务实例。
// 返回 domain.ToolService 接口，调用方不持有具体类型。
func NewToolService(store *confirm.Store, workdir string) domain.ToolService {
	return &ToolService{
		store:   store,
		workdir: workdir,
	}
}
