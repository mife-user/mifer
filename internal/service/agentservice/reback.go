package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// ListRebackEntries 返回当前会话中所有可回退的对话轮次
func (s *AgentService) ListRebackEntries(ctx context.Context) (*domain.RebackListResp, error) {
	var resp *domain.RebackListResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ListRebackEntries(ctx)
		return err
	})
	if err != nil {
		logger.Error("获取回退列表失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}

// Reback 回退到指定轮次的用户消息
func (s *AgentService) Reback(ctx context.Context, req *domain.RebackReq) (*domain.RebackResp, error) {
	var resp *domain.RebackResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.Reback(ctx, req)
		return err
	})
	if err != nil {
		logger.Error("回退对话失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
