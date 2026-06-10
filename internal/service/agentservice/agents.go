package agentservice

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// ListAgents 返回所有配置的 Agent 信息
func (s *AgentService) ListAgents(ctx context.Context) (*domain.AgentListResp, error) {
	var resp *domain.AgentListResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ListAgents(ctx)
		return err
	})
	if err != nil {
		logger.Error("获取Agent列表失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
