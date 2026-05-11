package agent

import (
	"context"
	"fmt"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/memory"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
)

type Humen struct {
	Agent  adk.Agent
	Memory *memory.Memory
}

func Init(c context.Context, config *conf.Config) (*Humen, error) {
	// 初始化LLM
	llm, err := llm.Init(c, config)
	if err != nil {
		logger.Error("init llm failed", logger.C(err))
		return nil, err
	}
	// 初始化聊天agent
	chatAgent, err := newChatAgent(c, llm)
	if err != nil {
		logger.Error("init chat agent failed", logger.C(err))
		return nil, err
	}
	// 初始化文件编辑agent
	editerAgent, err := newChatEditer(c, llm)
	if err != nil {
		logger.Error("init editer agent failed", logger.C(err))
		return nil, err
	}
	// 初始化agent
	agent, err := deep.New(c, &deep.Config{
		Name:         "Mifer",
		Description:  "管理员",
		Instruction:  " 你是一个智能助手，能够智能管理agent，调用其他agent，处理用户请求。",
		ChatModel:    llm.Model,
		SubAgents:    []adk.Agent{chatAgent, editerAgent},
		MaxIteration: 3,
	})
	if err != nil {
		logger.Error("init agent failed", logger.C(err))
		return nil, err
	}
	id, ok := c.Value("id").(string)
	if !ok {
		logger.Error("id is not string")
		return nil, fmt.Errorf("id is not string")
	}
	memory, err := memory.Init(config, id)
	if err != nil {
		logger.Error("init memory failed", logger.C(err))
		return nil, err
	}

	return &Humen{Agent: agent, Memory: memory}, nil
}
