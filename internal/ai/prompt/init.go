package prompt

import (
	"mifer/internal/ai/memory"
	"mifer/pkg/conf"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// New 使用默认系统提示创建 Prompty，自动加载 MIFER.md 文件拼接系统提示词
func New(m *memory.Memory) *Prompty {
	p := &Prompty{
		Memory:   m,
		Template: newDefaultTemplate(),
	}
	p.buildSystemPrompt()
	return p
}

// buildSystemPrompt 从 MIFER.md 文件构建系统提示词
// 顺序：用户级（CfgPath/MIFER.md）→ 项目级（workdir/.mifer/MIFER.md）→ 默认系统提示词
func (p *Prompty) buildSystemPrompt() {
	cfg := conf.GetConfig()
	var parts []string

	// 用户级 MIFER.md
	if content, ok := readMiferFile(filepath.Join(cfg.Path.CfgPath, "MIFER.md")); ok {
		parts = append(parts, content)
	}
	// 项目级 MIFER.md
	if content, ok := readMiferFile(filepath.Join(cfg.Path.Workdir, ".mifer", "MIFER.md")); ok {
		parts = append(parts, content)
	}
	// 默认系统提示词
	parts = append(parts, defaultSystemPrompt)

	p.SystemPrompt = strings.Join(parts, "\n")
}

// readMiferFile 读取并 trim 指定路径的 MIFER.md 文件
func readMiferFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", false
	}
	return content, true
}

// newDefaultTemplate 构建默认 ChatTemplate：
//
//	System: {system_prompt}
//	MessagesPlaceholder: {history}（对话历史动态插入）
//	User: {query}
func newDefaultTemplate() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("{system_prompt}"),
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
