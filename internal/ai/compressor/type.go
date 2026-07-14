package compressor

import (
	"mifer/internal/ai/llm"
	"mifer/internal/ai/offload"
	"mifer/pkg/skill"
)

// Compressor 上下文压缩器，负责在对话上下文超限时生成摘要并替换记忆。
// 支持三层记忆模型：Layer1 完整保留、Layer2 精简（保留 ToolCall + 截断结果）、Layer3 摘要。
type Compressor struct {
	registry     *llm.Registry     // 模型注册中心，用于获取压缩模型
	skillManager *skill.Manager    // 技能管理器，用于获取压缩提示词模板
	offloader    offload.Offloader // offload 后端，用于存储大工具结果
	recentRounds int               // N: 完整保留的最近轮数
	slimRounds   int               // M: 精简保留的中间轮数
}
