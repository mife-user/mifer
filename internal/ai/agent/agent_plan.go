package agent

import (
	"context"

	"mifer/internal/ai/llm"
	"mifer/internal/ai/tools"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
)

// createPlanAgent 创建计划制定 Agent（只读工具），失败时静默返回空值。
func (h *Humen) createPlanAgent(ctx context.Context, reg *llm.Registry) (adk.Agent, AgentInfo) {
	agentModel := getBackendModel(reg, "plan_agent")
	if agentModel == nil {
		return nil, AgentInfo{}
	}

	pa, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlanAgent",
		Description: "计划制定助手，只能查看文件和搜索，不可写入或执行",
		Instruction: PlanInstruction,
		Model:       agentModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               tools.ReadOnlyTools(h.ragSvc),
				ToolCallMiddlewares: []compose.ToolMiddleware{h.errorMw, confirmMiddleware, h.persistenceMw},
			},
		},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, AgentInfo{}
	}
	return pa, AgentInfo{Name: "PlanAgent", ModelBackend: getBackendName(conf.GetConfig(), "plan_agent", reg), Description: "计划制定助手，只读分析"}
}
