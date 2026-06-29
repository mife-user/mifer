package agentservice

import (
	"context"

	"mifer/internal/domain"
)

// BackendStatus 返回当前各后端的加载状态
func (s *AgentService) BackendStatus(ctx context.Context) (*domain.BackendStatusResp, error) {
	return s.Executor.BackendStatus(ctx)
}
