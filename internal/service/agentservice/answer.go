package agentservice

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// Answer 处理用户问答回复，直接委托给 executor。
func (s *AgentService) Answer(ctx context.Context, req *domain.AnswerReq) (*domain.AnswerResp, error) {
	resp, err := s.Executor.Answer(ctx, req)
	if err != nil {
		logger.Error("处理问答回复失败", logger.C(err))
		return nil, err
	}
	return resp, nil
}
