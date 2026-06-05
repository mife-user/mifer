package executor

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// TokenUsage token 累计用量统计
type TokenUsage struct {
	Prompt     int // 输入 token
	Completion int // 输出 token
	Total      int // 合计 token
	Cached     int // 缓存命中 token
	Reasoning  int // 推理 token
}

func initTokenUsage() *TokenUsage {
	return &TokenUsage{
		Prompt:     0,
		Completion: 0,
		Total:      0,
		Cached:     0,
		Reasoning:  0,
	}
}

// accumulate 从 schema.Message 中累加 token 用量，返回是否有新数据
func (t *TokenUsage) accumulate(msg *schema.Message) bool {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return false
	}
	usage := msg.ResponseMeta.Usage
	t.Prompt += usage.PromptTokens
	t.Completion += usage.CompletionTokens
	t.Total += usage.TotalTokens
	t.Cached += usage.PromptTokenDetails.CachedTokens
	t.Reasoning += usage.CompletionTokensDetails.ReasoningTokens
	return true
}

// send 发送 token 事件到回调
func (t *TokenUsage) send(callback func(event, content string) error) error {
	payload := fmt.Sprintf("%d\x00%d\x00%d\x00%d\x00%d",
		t.Prompt, t.Completion, t.Total, t.Cached, t.Reasoning)
	return callback("token", payload)
}

// reset 重置所有计数为零
func (t *TokenUsage) reset() {
	t.Prompt = 0
	t.Completion = 0
	t.Total = 0
	t.Cached = 0
	t.Reasoning = 0
}
