package agent

import (
	"context"

	"mifer/internal/ai/llm"
	"mifer/internal/ai/tools"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"mifer/pkg/skill"
	"mifer/qq"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// createCustomAgents 根据配置文件 agents 段创建自定义 Agent，注册到 skillHub 并返回子 Agent 列表及元数据。
func (h *Humen) createCustomAgents(ctx context.Context, reg *llm.Registry) ([]adk.Agent, []AgentInfo, error) {
	var subagents []adk.Agent
	var infos []AgentInfo

	for _, cfg := range conf.GetConfig().Agents {
		agentTools, err := tools.NewWithName(cfg.Tools, h.ragSvc)
		if err != nil {
			logger.Error("创建自定义Agent工具失败", logger.S("agent", cfg.Name), logger.C(err))
			return nil, nil, err
		}
		agentModel := reg.Get(cfg.Model)
		if agentModel == nil {
			agentModel = reg.First()
			if agentModel == nil {
				logger.Warn("自定义Agent无可用模型", logger.S("agent", cfg.Name))
				continue
			}
			logger.Warn("自定义Agent指定后端不存在，使用默认后端", logger.S("agent", cfg.Name), logger.S("backend", cfg.Model))
		}
		extraAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name:        cfg.Name,
			Description: cfg.Description,
			Instruction: cfg.Instruction,
			Model:       agentModel,
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools:               agentTools,
					ToolCallMiddlewares: []compose.ToolMiddleware{h.errorMw, confirmMiddleware},
				},
			},
			MaxIterations: 100,
		})
		if err != nil {
			logger.Error("创建自定义Agent失败", logger.S("name", cfg.Name), logger.C(err))
			return nil, nil, err
		}
		subagents = append(subagents, extraAgent)
		h.skillHub.Register(cfg.Name, extraAgent)
		infos = append(infos, AgentInfo{Name: cfg.Name, ModelBackend: cfg.Model, Description: cfg.Description, Tools: resolveToolNames(ctx, agentTools)})
	}
	return subagents, infos, nil
}

// createMiferAgent 创建 Mifer 编排器（deep.New），注入全部工具及自定义子 Agent。
func (h *Humen) createMiferAgent(ctx context.Context, reg *llm.Registry, subagents []adk.Agent) (adk.Agent, AgentInfo, error) {
	orchTools := []tool.BaseTool{skill.NewSkillTool(h.SkillManager, h.skillHub)}
	for _, t := range mcpToBaseTools(h.MCPManager.GetToolsForAgent("Mifer")) {
		orchTools = append(orchTools, t)
	}
	for _, t := range tools.WebTools() {
		orchTools = append(orchTools, t)
	}
	for _, t := range tools.QQTools(func() qq.Sender { return nil }) {
		orchTools = append(orchTools, t)
	}
	for _, t := range tools.FileTools() {
		orchTools = append(orchTools, t)
	}
	for _, t := range tools.CommandTools() {
		orchTools = append(orchTools, t)
	}
	if h.skillHub.HasAny() {
		for _, t := range tools.ParallelDispatch(h.skillHub) {
			orchTools = append(orchTools, t)
		}
	}
	for _, t := range tools.KnowledgeTools(h.ragSvc) {
		orchTools = append(orchTools, t)
	}

	agentModel := getBackendModel(reg, "mifer")
	if agentModel == nil {
		logger.Error("Mifer Agent 无可用模型")
		return nil, AgentInfo{}, nil
	}

	agent, err := deep.New(ctx, &deep.Config{
		Name:        "Mifer",
		Description: "Mifer 智能助手，具备文件操作、命令执行、知识库管理、网页搜索和 QQ 消息等完整能力",
		Instruction: miferInstruction,
		ChatModel:   agentModel,
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true,
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               orchTools,
				ToolCallMiddlewares: []compose.ToolMiddleware{h.errorMw, confirmMiddleware},
			},
		},
		SubAgents:    subagents,
		MaxIteration: 100,
	})
	if err != nil {
		logger.Error("init agent failed", logger.C(err))
		return nil, AgentInfo{}, err
	}

	info := AgentInfo{Name: "Mifer", ModelBackend: getBackendName(conf.GetConfig(), "mifer", reg), Description: "Mifer 智能助手，具备文件操作、命令执行、知识库管理、网页搜索和 QQ 消息等完整能力", Tools: resolveToolNames(ctx, orchTools)}
	return agent, info, nil
}

// getBackendName 获取 Agent 配置的后端名称，用于 AgentInfo
func getBackendName(cfg *conf.Config, agentName string, reg *llm.Registry) string {
	if name, ok := cfg.Ai.AgentBackends[agentName]; ok && name != "" {
		return name
	}
	return reg.FirstKey()
}

// mcpToBaseTools 将 []tool.InvokableTool 转为 []tool.BaseTool。
func mcpToBaseTools(invokable []tool.InvokableTool) []tool.BaseTool {
	var result []tool.BaseTool
	for _, t := range invokable {
		result = append(result, t)
	}
	return result
}
