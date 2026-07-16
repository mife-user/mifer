package agent

import (
	"context"

	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
)

// AgentInfo 保存 Init() 期间记录的 agent 元数据
type AgentInfo struct {
	Name         string   // agent 名称，如 "MiEditer"、"Mifer"
	ModelBackend string   // 配置后端 key，如 "main"、"fast-model"
	Description  string   // agent 描述
	Tools        []string // 已解析的工具名列表
}

// resolveToolNames 提取每个工具的 Info().Name，用于记录 agent 的工具集
func resolveToolNames(ctx context.Context, tools []tool.BaseTool) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		info, err := t.Info(ctx)
		if err != nil || info == nil {
			logger.Warn(ctx, "获取工具信息失败，跳过该工具", logger.C(err))
			continue
		}
		names = append(names, info.Name)
	}
	return names
}
