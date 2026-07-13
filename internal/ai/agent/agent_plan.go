package agent

import (
	"context"

	"mifer/internal/ai/llm"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// createPlanAgent 创建计划制定 Agent（只读工具，sonnet 模型），失败时静默返回空值。
func (h *Humen) createPlanAgent(ctx context.Context, reg *llm.Registry, mmModel model.BaseChatModel) (adk.Agent, AgentInfo) {
	pa, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlanAgent",
		Description: "计划制定助手，只能查看文件和搜索，不可写入或执行",
		Instruction: PlanInstruction,
		Model:       reg.Get("sonnet"),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               tools.ReadOnlyTools(mmModel, h.ragSvc),
				ToolCallMiddlewares: []compose.ToolMiddleware{h.errorMw, confirmMiddleware},
			},
		},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, AgentInfo{}
	}
	return pa, AgentInfo{Name: "PlanAgent", ModelBackend: "sonnet", Description: "计划制定助手，只读分析"}
}
