package agent

import (
	"context"

	"mifer/internal/ai/llm"

	"github.com/cloudwego/eino/adk"
)

// createQQAgent 创建 QQ 通道专用 Agent（无工具，纯文本对话），失败时静默返回空值。
func (h *Humen) createQQAgent(ctx context.Context, reg *llm.Registry) (adk.Agent, AgentInfo) {
	qa, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "MiQQ",
		Description:   "QQ 消息助手，直接回复用户问题",
		Instruction:   qqInstruction,
		Model:         reg.Get("default"),
		MaxIterations: 1,
	})
	if err != nil {
		return nil, AgentInfo{}
	}
	return qa, AgentInfo{Name: "MiQQ", ModelBackend: "default", Description: "QQ 通道专用助手，纯文本对话"}
}
