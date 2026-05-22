package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// Build 组装完整 prompt：RAG 检索 → 模板变量替换 → 返回消息列表
// query 为用户本轮输入，同时作为 RAG 检索的查询文本和模板中 {query} 变量的值
func (p *Prompty) Build(ctx context.Context, query string) ([]*schema.Message, error) {
	contextStr := ""
	if p.RAGService != nil {
		docs, err := p.RAGService.Retrieve(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("RAG检索失败: %w", err)
		}
		contextStr = formatDocs(docs)
	}

	// 若未配置模板，回退到手工拼接（保持向后兼容）
	if p.Template == nil {
		return p.buildLegacy(contextStr, query), nil
	}

	return p.Template.Format(ctx, map[string]any{
		"system_prompt": p.SystemPrompt,
		"context":       contextStr,
		"history":       p.Memory.Messages,
		"query":         query,
	})
}

// buildLegacy 手工拼接消息，用于模板不可用时的回退路径
func (p *Prompty) buildLegacy(contextStr, query string) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(p.Memory.Messages)+2)
	if p.SystemPrompt != "" {
		content := p.SystemPrompt
		if contextStr != "" {
			content += "\n\n## 参考知识库\n" + contextStr
		}
		msgs = append(msgs, schema.SystemMessage(content))
	}
	msgs = append(msgs, p.Memory.Messages...)
	msgs = append(msgs, schema.UserMessage(query))
	return msgs
}

// formatDocs 将检索到的文档格式化为上下文字符串
func formatDocs(docs []*schema.Document) string {
	if len(docs) == 0 {
		return "暂无相关知识库内容"
	}
	var sb strings.Builder
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("【文档%d】%s\n", i+1, doc.Content))
	}
	return sb.String()
}
