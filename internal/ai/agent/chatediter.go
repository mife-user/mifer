package agent

import (
	"context"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
)

// newChatEditer 创建文件编辑agent，负责文件读写操作
func newChatEditer(c context.Context, llm *llm.LLM) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        "MiEditer",
		Description: "专门处理文件读写操作的agent，能够安全地读取和写入本地文件",
		Instruction: " 你是文件编辑专家，负责处理用户的文件读写请求。读取文件时注意行数限制，写入文件时确认路径安全。",
		Model:       llm.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools.AllTools(),
			},
		},
		MaxIterations: 3,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
