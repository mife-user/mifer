package executor

import (
	"mifer/pkg/conf"
	"mifer/pkg/logger"
)

// checkToolCompression 检查是否需要压缩旧轮次的工具调用消息。
//
// 当配置 tool_keep_rounds > 0 且记忆中的工具调用轮次超过该值时，
// 将超过阈值的旧工具调用替换为摘要消息。
// 压缩后的消息由 CompressToolHistory 直接持久化，调用方无需再调用 Save。
func (e *Executor) checkToolCompression() {
	cfg := conf.GetConfig().Ai.Context
	if cfg.ToolKeepRounds <= 0 {
		return
	}

	e.Humen.Prompt.Memory.CompressToolHistory(cfg.ToolKeepRounds)
	logger.Debug("工具调用压缩检查完成", logger.I("keep_rounds", cfg.ToolKeepRounds))
}
