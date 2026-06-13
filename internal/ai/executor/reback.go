package executor

import (
	"context"
	"fmt"
	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// ListRebackEntries 返回当前会话中所有可回退的对话轮次
func (e *Executor) ListRebackEntries(c context.Context) (*domain.RebackListResp, error) {
	entries, err := e.Humen.Prompt.Memory.ListRebackEntries()
	if err != nil {
		logger.Error("获取回退列表失败", logger.C(err))
		return nil, err
	}
	domainEntries := make([]domain.RebackEntry, len(entries))
	for i, entry := range entries {
		domainEntries[i] = domain.RebackEntry{
			Index:   entry.Index,
			Summary: entry.Summary,
		}
	}
	return &domain.RebackListResp{Entries: domainEntries}, nil
}

// Reback 回退到指定轮次之前
func (e *Executor) Reback(c context.Context, req *domain.RebackReq) (*domain.RebackResp, error) {
	summary, content, err := e.Humen.Prompt.Memory.Reback(req.Index)
	if err != nil {
		logger.Error("执行回退失败", logger.C(err))
		return nil, err
	}
	return &domain.RebackResp{
		Summary: summary,
		Content: content,
		Message: fmt.Sprintf("已回退到第%d轮对话: %s", req.Index, summary),
	}, nil
}
