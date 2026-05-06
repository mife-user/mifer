package agent

import (
	"context"
	"mifer/internal/ai/llm"

	"github.com/cloudwego/eino/adk"
)

// newChatAgent 创建一个聊天agent
func newChatAgent(c context.Context, llm *llm.LLM) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:          "MiTalker",
		Description:   "与用户进行生活交流的agent",
		Instruction:   " 与用户进行生活交流。",
		Model:         llm.Model,
		MaxIterations: 3,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
