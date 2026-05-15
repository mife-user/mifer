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
	// 初始化文档摘要agent
	summarizerAgent, err := newSummarizer(c, llm)
	if err != nil {
		logger.Error("init summarizer agent failed", logger.C(err))
		return nil, err
	}
	// 初始化计划编写agent
	plannerAgent, err := newPlanner(c, llm)
	if err != nil {
		logger.Error("init planner agent failed", logger.C(err))
		return nil, err
	}
	// 初始化终端命令agent
	commanderAgent, err := newCommander(c, llm)
	if err != nil {
		logger.Error("init commander agent failed", logger.C(err))
		return nil, err
	}
	// 初始化安全审计agent
	auditorAgent, err := newAuditor(c, llm)
	if err != nil {
		logger.Error("init auditor agent failed", logger.C(err))
		return nil, err
	}
	// 初始化编排器agent
	agent, err := deep.New(c, &deep.Config{
		Name:        "Mifer",
		Description: "智能任务编排器，根据用户请求自动选择最合适的专家Agent处理任务",
		Instruction: " 你是Mifer智能助手的管理员，负责分析用户请求并调度合适的专家Agent。\n\n你可以调用的专家Agent：\n- MiTalker：日常对话交流\n- MiEditer：文件读取、写入、创建\n- MiSummarizer：文档阅读与摘要总结\n- MiPlanner：项目计划与方案编写\n- MiCommander：安全执行终端命令\n- MiAuditor：代码与配置安全审计\n\n工作原则：\n1. 先理解用户意图，再选择合适的Agent\n2. 复杂任务可串联多个Agent协作完成\n3. 涉及安全操作时优先咨询MiAuditor\n4. 回复用户时使用中文，简洁清晰",
		ChatModel:   llm.Model,
		SubAgents:   []adk.Agent{chatAgent, editerAgent, summarizerAgent, plannerAgent, commanderAgent, auditorAgent},
		MaxIteration: 5,
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
