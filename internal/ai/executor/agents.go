package executor

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
)

// ListAgents 返回所有配置的 Agent 信息
func (e *Executor) ListAgents(ctx context.Context) (*domain.AgentListResp, error) {
	cfg := conf.GetConfig()
	var agents []domain.AgentInfo
	for _, a := range e.Humen.AgentInfos {
		backend, ok := cfg.Ai.Backends[a.ModelBackend]
		provider := ""
		model := ""
		if ok {
			provider = backend.Provider
			model = backend.Model
		}
		tools := a.Tools
		if tools == nil {
			tools = []string{}
		}
		agents = append(agents, domain.AgentInfo{
			Name:         a.Name,
			ModelBackend: a.ModelBackend,
			Provider:     provider,
			Model:        model,
			Description:  a.Description,
			Tools:        tools,
		})
	}

	if agents == nil {
		agents = []domain.AgentInfo{}
	}

	logger.Debug(ctx, "Agent列表查询完成")
	return &domain.AgentListResp{Agents: agents}, nil
}
