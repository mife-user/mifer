package agent

import (
	"context"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/memory"
	"mifer/internal/ai/prompt"
	"mifer/internal/ai/rag"
	"mifer/internal/ai/tools"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
)

type Humen struct {
	Agent    adk.Agent
	Prompt   *prompt.Prompty
	Registry *llm.Registry
}

func Init(c context.Context) (*Humen, error) {
	// 初始化LLM注册中心
	reg, err := llm.InitRegistry(c)
	if err != nil {
		logger.Error("初始化LLM注册中心失败", logger.C(err))
		return nil, err
	}
	mmModel := reg.Get("multi_modal")

	// RAG 懒加载服务，无网络调用，即时返回，Qdrant 连接推迟到首次工具调用
	ragSvc := rag.NewLazyService(c)

	// 初始化聊天agent（haiku — 快速响应）
	chatAgent, err := newChatAgent(c, reg.Get("haiku"))
	if err != nil {
		logger.Error("init chat agent failed", logger.C(err))
		return nil, err
	}
	// 初始化文件编辑agent（sonnet — 均衡）
	editerAgent, err := newChatEditer(c, reg.Get("sonnet"), mmModel)
	if err != nil {
		logger.Error("init editer agent failed", logger.C(err))
		return nil, err
	}
	// 初始化文档摘要agent（sonnet — 均衡），注入知识库工具
	summarizerAgent, err := newSummarizer(c, reg.Get("sonnet"), mmModel, tools.KnowledgeTools(ragSvc))
	if err != nil {
		logger.Error("init summarizer agent failed", logger.C(err))
		return nil, err
	}
	// 初始化计划编写agent（opus — 最强推理）
	plannerAgent, err := newPlanner(c, reg.Get("opus"), mmModel)
	if err != nil {
		logger.Error("init planner agent failed", logger.C(err))
		return nil, err
	}
	// 初始化终端命令agent（sonnet — 均衡）
	commanderAgent, err := newCommander(c, reg.Get("sonnet"))
	if err != nil {
		logger.Error("init commander agent failed", logger.C(err))
		return nil, err
	}
	// 初始化安全审计agent（opus — 最强推理）
	auditorAgent, err := newAuditor(c, reg.Get("opus"), mmModel)
	if err != nil {
		logger.Error("init auditor agent failed", logger.C(err))
		return nil, err
	}
	// 初始化编排器agent（default — 调度主脑）
	agent, err := deep.New(c, &deep.Config{
		Name:        "Mifer",
		Description: "智能任务编排器，根据用户请求自动选择最合适的专家Agent处理任务",
		Instruction: " 你是Mifer智能助手的管理员，负责分析用户请求并调度合适的专家Agent。\n\n你可以调用的专家Agent：\n- MiTalker：日常对话交流\n- MiEditer：文件读取、写入、创建\n- MiSummarizer：文档阅读、摘要总结与知识库管理（支持知识库检索和文档入库）\n- MiPlanner：项目计划与方案编写\n- MiCommander：安全执行终端命令\n- MiAuditor：代码与配置安全审计\n\n工作原则：\n1. 先理解用户意图，再选择合适的Agent\n2. 复杂任务可串联多个Agent协作完成\n3. 涉及安全操作时优先咨询MiAuditor\n4. 回复用户时使用中文，简洁清晰",
		ChatModel:   reg.Get("default"),
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true, // 转发子Agent内部事件到父级事件流，使TUI侧边栏可显示子Agent及工具调用
		},
		SubAgents:    []adk.Agent{chatAgent, editerAgent, summarizerAgent, plannerAgent, commanderAgent, auditorAgent},
		MaxIteration: 0,
	})
	if err != nil {
		logger.Error("init agent failed", logger.C(err))
		return nil, err
	}
	id, ok := c.Value("id").(string)
	if !ok {
		logger.Error("id is not string")
		return nil, errorer.New(errorer.ErrIDNotString)
	}
	mem, err := memory.Init(id)
	if err != nil {
		logger.Error("init memory failed", logger.C(err))
		return nil, err
	}

	prompty := prompt.New(mem)
	return &Humen{Agent: agent, Prompt: prompty, Registry: reg}, nil
}
