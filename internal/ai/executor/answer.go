package executor

import (
	"context"

	"mifer/internal/ai/question"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
)

// Answer 处理用户问答回复，解除 ask_user 工具的阻塞等待。
func (e *Executor) Answer(ctx context.Context, req *domain.AnswerReq) (*domain.AnswerResp, error) {
	if req.ID == "" {
		return nil, errorer.New("问题ID不能为空")
	}

	store := e.Humen.QuestionStore
	if store == nil {
		return nil, errorer.New("问题存储未初始化")
	}

	_, ok := store.Get(req.ID)
	if !ok {
		logger.Warn("待回答问题不存在或已超时", logger.S("id", req.ID))
		return &domain.AnswerResp{Resolved: false, Message: "问题不存在或已超时"}, nil
	}

	store.Resolve(req.ID, question.QuestionResult{
		Answer:       req.Answer,
		IsSupplement: req.IsSupplement,
	})

	logger.Debug("问答回复已解析",
		logger.S("id", req.ID),
		logger.S("answer", req.Answer))

	return &domain.AnswerResp{Resolved: true, Message: "回答已提交"}, nil
}
