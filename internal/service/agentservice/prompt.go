package agentservice

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
	"mifer/pkg/task"
)

// GetPrompt 获取当前系统提示词
func (s *AgentService) GetPrompt(ctx context.Context) (*domain.PromptResp, error) {
	var resp *domain.PromptResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.GetPrompt(ctx)
		return err
	})
	if err != nil {
		logger.Error("获取提示词失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}

// SetPrompt 设置自定义系统提示词
func (s *AgentService) SetPrompt(ctx context.Context, req *domain.PromptReq) (*domain.PromptResp, error) {
	var resp *domain.PromptResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.SetPrompt(ctx, req)
		return err
	})
	if err != nil {
		logger.Error("设置提示词失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}

// ResetPrompt 重置为默认系统提示词
func (s *AgentService) ResetPrompt(ctx context.Context) (*domain.PromptResp, error) {
	var resp *domain.PromptResp
	var err error
	err = task.Do(ctx, func() error {
		resp, err = s.Executor.ResetPrompt(ctx)
		return err
	})
	if err != nil {
		logger.Error("重置提示词失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
