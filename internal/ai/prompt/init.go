package prompt

import (
	"mifer/internal/ai/memory"
	"mifer/internal/ai/rag"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// NewWithRAG 使用默认系统提示 + RAG 服务创建 Prompty
func NewWithRAG(m *memory.Memory, ragSvc *rag.Service) *Prompty {
	return NewWithPromptAndRAG(m, defaultSystemPrompt, ragSvc)
}

// NewWithPromptAndRAG 使用自定义系统提示 + RAG 服务创建 Prompty
func NewWithPromptAndRAG(m *memory.Memory, sysPrompt string, ragSvc *rag.Service) *Prompty {
	return &Prompty{
		Memory:       m,
		SystemPrompt: sysPrompt,
		Template:     newDefaultTemplate(),
		RAGService:   ragSvc,
	}
}

// newDefaultTemplate 构建默认 ChatTemplate：
//
//	System: {system_prompt} + RAG {context}
//	MessagesPlaceholder: {history}（对话历史动态插入）
//	User: {query}
func newDefaultTemplate() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("{system_prompt}\n\n## 参考知识库\n{context}"),
		schema.MessagesPlaceholder("history", false),
		schema.UserMessage("{query}"),
	)
}

// 默认系统提示词：告知 AI 可以在任务完成后自行退出，避免无限循环
const defaultSystemPrompt = `你是Mifer智能助手。
	【重要】工作原则：
	- 当任务目标已经达成时，输出最终结果并结束对话，无需继续调用工具、再次确认或询问用户是否需要继续。
	- 如果用户的请求已经得到充分回应，直接停止，不需要额外的工具调用或循环检查。
	- 避免无意义的重复工具调用：如果同一个工具已经调用并成功返回结果，不要再次调用。
	- 在给出最终答案后，不要追问"还有什么可以帮您的吗？"，直接结束即可。`
