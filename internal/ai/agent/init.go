package agent

import (
	"context"
	"fmt"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/memory"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type Humen struct {
	Agent  *adk.Agent
	Memory *memory.Memory
}

func Init(c context.Context, config *conf.Config) (*Humen, error) {
	// 初始化LLM
	llm, err := llm.Init(c, config)
	if err != nil {
		return nil, err
	}
	// 初始化聊天agent
	chatAgent, err := newChatAgent(c, llm)
	if err != nil {
		return nil, err
	}
	// 初始化agent
	agent, err := deep.New(c, &deep.Config{
		Name:        "Mifer",
		Description: "管理员",
		Instruction: " 你是一个智能助手，能够智能管理agent，调用其他agent，处理用户请求。",
		ChatModel:   llm.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{},
			},
		},
		SubAgents:    []adk.Agent{chatAgent},
		MaxIteration: 3,
	})
	if err != nil {
		return nil, err
	}
	id, ok := c.Value("id").([]byte)
	if !ok {
		return nil, fmt.Errorf("id is not []byte")
	}
	memory, err := memory.Init(config, id)
	if err != nil {
		return nil, err
	}

	return &Humen{Agent: &agent, Memory: memory}, nil
}
