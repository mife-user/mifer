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
		logger.Error(c, "获取回退列表失败", logger.C(err))
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
	// 回退前获取当前总轮次数（用于后续快照计算）
	totalRounds := e.Humen.Prompt.Memory.CountUserMessages()
	summary, content, err := e.Humen.Prompt.Memory.Reback(req.Index)
	if err != nil {
		logger.Error(c, "执行回退失败", logger.C(err))
		return nil, err
	}
	// 从文件快照恢复（需在配置中启用 snapshot_enabled）
	if e.Snapshot != nil {
		targetRound := req.Index - 1 // 恢复到目标轮次之前的状态
		logger.Debug(c, "开始从快照恢复文件",
			logger.I("reqIndex", req.Index),
			logger.I("targetRound", targetRound),
			logger.I("totalRounds", totalRounds),
		)
		if err := e.Snapshot.RestoreToRound(targetRound); err != nil {
			logger.Warn(c, "从快照恢复文件失败", logger.C(err))
		} else {
			logger.Info(c, "从快照恢复文件成功", logger.I("targetRound", targetRound))
		}
		// 删除被回退的后续快照
		for i := req.Index; i <= totalRounds; i++ {
			logger.Debug(c, "删除被回退的快照轮次", logger.I("round", i))
			e.Snapshot.RemoveRound(i)
		}
	}
	return &domain.RebackResp{
		Summary: summary,
		Content: content,
		Message: fmt.Sprintf("已回退到第%d轮对话: %s", req.Index, summary),
	}, nil
}
