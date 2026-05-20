package prompt

import "mifer/internal/ai/memory"

// Prompty 包装 Memory，负责组装完整 prompt（系统提示 + 对话历史）
type Prompty struct {
	Memory       *memory.Memory // 底层对话记忆
	SystemPrompt string         // 系统提示词，Build() 时会 prepend 到消息列表最前面
}

// New 使用默认系统提示创建 Prompty
func New(m *memory.Memory) *Prompty {
	return &Prompty{
		Memory:       m,
		SystemPrompt: defaultSystemPrompt,
	}
}

// NewWithPrompt 使用自定义系统提示创建 Prompty
func NewWithPrompt(m *memory.Memory, sysPrompt string) *Prompty {
	return &Prompty{
		Memory:       m,
		SystemPrompt: sysPrompt,
	}
}

// 默认系统提示词：告知 AI 可以在任务完成后自行退出，避免无限循环
const defaultSystemPrompt = `你是Mifer智能助手。
【重要】工作原则：
- 当任务目标已经达成时，输出最终结果并结束对话，无需继续调用工具、再次确认或询问用户是否需要继续。
- 如果用户的请求已经得到充分回应，直接停止，不需要额外的工具调用或循环检查。
- 避免无意义的重复工具调用：如果同一个工具已经调用并成功返回结果，不要再次调用。
- 在给出最终答案后，不要追问"还有什么可以帮您的吗？"，直接结束即可。`
