package compressor

import (
	"mifer/internal/ai/llm"
	"mifer/pkg/skill"
)

// NewCompressor 创建上下文压缩器
func NewCompressor(registry *llm.Registry, skillMgr *skill.Manager) *Compressor {
	return &Compressor{
		registry:     registry,
		skillManager: skillMgr,
	}
}
