package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// MCPStatus 返回所有 MCP Server 的连接状态
func (s *AgentService) MCPStatus(ctx context.Context) (*domain.MCPStatusResp, error) {
	var resp *domain.MCPStatusResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.MCPStatus(ctx)
		return err
	})
	if err != nil {
		logger.Error(ctx, "获取MCP状态失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
