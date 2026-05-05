package agent

import (
	"context"
	"mifer/internal/ai/llm"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type Agent struct {
	Agent *adk.ChatModelAgent
}

func Init(c context.Context, config *conf.Config) (*Agent, error) {
	// 初始化LLM
	llm, err := llm.Init(c, config)
	if err != nil {
		return nil, err
	}
	// 初始化agent
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "Mifer",
		Description: "主智能助手",
		Instruction: "你是go语言的智能助手，能够智能管理agent，调用其他agent，处理用户请求。",
		Model:       llm.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{},
			},
		},
		MaxIterations: 3,
	})
	if err != nil {
		return nil, err
	}

	return &Agent{Agent: agent}, err
}
