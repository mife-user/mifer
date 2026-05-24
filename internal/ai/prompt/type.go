package prompt

import (
	"mifer/internal/ai/memory"

	"github.com/cloudwego/eino/components/prompt"
)

// Prompty 包装 Memory，负责通过 Eino 模板引擎组装完整 prompt（系统提示 + 对话历史 + 用户输入）
type Prompty struct {
	Memory       *memory.Memory              // 底层对话记忆
	SystemPrompt string                      // 系统提示词
	Template     *prompt.DefaultChatTemplate // Eino 模板引擎
}
