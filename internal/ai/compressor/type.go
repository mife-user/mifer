package compressor

import (
	"mifer/internal/ai/llm"
	"mifer/pkg/skill"
)

// Compressor 上下文压缩器，负责在对话上下文超限时生成摘要并替换记忆
type Compressor struct {
	registry     *llm.Registry  // 模型注册中心，用于获取压缩模型
	skillManager *skill.Manager // 技能管理器，用于获取压缩提示词模板
}
