package executor

import (
	"context"
	"strings"

	"mifer/internal/domain"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

func (e *Executor) LoadMemory(c context.Context, req *domain.MemoryReq) (*domain.MemoryResp, error) {
	// 根据请求ID决定数据源：当前会话走内存缓存，其他会话从磁盘加载
	var msgs []*schema.Message
	if req.ID == e.Humen.Prompt.Memory.GetCurrentID() {
		msgs = e.Humen.Prompt.Memory.Messages()
	} else {
		var err error
		msgs, err = e.Humen.Prompt.Memory.LoadByID(req.ID)
		if err != nil {
			logger.Error("加载记忆失败", logger.C(err))
			return nil, err
		}
	}

	var sb strings.Builder
	for _, msg := range msgs {
		// 跳过工具消息（不向用户展示），保持 /viewmemory 仅显示对话内容
		if msg.Role == schema.Tool {
			continue
		}
		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			continue
		}
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
