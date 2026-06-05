package executor

import (
	"context"

	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// Compact 手动触发上下文压缩，通过 REST 接口暴露给 CLI /compact 命令
func (e *Executor) Compact(ctx context.Context) (*domain.CompactResp, error) {
	// 重置自动压缩标记（手动压缩后无需再次自动压缩）
	e.needsCompression = false

	// 调用压缩器，callback 为空操作（非流式场景无需通知 TUI）
	err := e.Compressor.Compress(
		ctx,
		e.Humen.Prompt.Memory,
		e.Humen.Prompt.SystemPrompt,
		e.Token.Prompt,
		func(event, content string) error { return nil },
	)
	if err != nil {
		logger.Error("手动压缩上下文失败", logger.C(err))
		return nil, err
	}

	logger.Info("手动压缩上下文完成")
	return &domain.CompactResp{Message: "上下文压缩完成"}, nil
}
