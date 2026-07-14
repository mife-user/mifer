package compressor

import (
	"mifer/internal/ai/llm"
	"mifer/internal/ai/offload"
	"mifer/pkg/skill"
)

// NewCompressor 创建上下文压缩器
func NewCompressor(registry *llm.Registry, skillMgr *skill.Manager, offloader offload.Offloader, recentRounds, slimRounds int) *Compressor {
	return &Compressor{
		registry:     registry,
		skillManager: skillMgr,
		offloader:    offloader,
		recentRounds: recentRounds,
		slimRounds:   slimRounds,
	}
}
