package executor

import (
	"context"
	"mifer/internal/domain"
)

// GetPrompt 获取当前系统提示词
func (e *Executor) GetPrompt(ctx context.Context) (*domain.PromptResp, error) {
	return &domain.PromptResp{
		Prompt: e.Humen.Prompt.GetSystemPrompt(),
	}, nil
}

// SetPrompt 设置自定义系统提示词
func (e *Executor) SetPrompt(ctx context.Context, req *domain.PromptReq) (*domain.PromptResp, error) {
	e.Humen.Prompt.SetSystemPrompt(req.Prompt)
	return &domain.PromptResp{
		Prompt: e.Humen.Prompt.GetSystemPrompt(),
	}, nil
}

// ResetPrompt 重置为默认系统提示词
func (e *Executor) ResetPrompt(ctx context.Context) (*domain.PromptResp, error) {
	e.Humen.Prompt.ResetSystemPrompt()
	return &domain.PromptResp{
		Prompt: e.Humen.Prompt.GetSystemPrompt(),
	}, nil
}
