package executor

import (
	"context"
	"mifer/internal/domain"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func (e *Executor) LoadMemory(c context.Context, req *domain.MemoryReq) (*domain.MemoryResp, error) {
	var sb strings.Builder
	for _, msg := range e.Humen.Prompt.Memory.Messages {
		sb.WriteString("[")
		sb.WriteString(roleToChinese(msg.Role))
		sb.WriteString("]: ")
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}
	return &domain.MemoryResp{Memory: sb.String()}, nil
}

// roleToChinese 将角色类型转换为中文显示名称
func roleToChinese(role schema.RoleType) string {
	switch role {
	case schema.User:
		return "用户"
	case schema.Assistant:
		return "助手"
	case schema.System:
		return "系统"
	case schema.Tool:
		return "工具"
	default:
		return string(role)
	}
}
