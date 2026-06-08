package agent

import (
	"context"
	"fmt"
	"strings"

	"mifer/internal/ai/rag"
	"mifer/internal/ai/tools/commandexecutor"
	"mifer/internal/ai/tools/filecreator"
	"mifer/internal/ai/tools/filereader"
	"mifer/internal/ai/tools/fileviewer"
	"mifer/internal/ai/tools/filewriter"
	"mifer/internal/ai/tools/imagegenerator"
	"mifer/internal/ai/tools/knowledgesearch"
	"mifer/internal/ai/tools/knowledgestore"
	"mifer/internal/ai/tools/webfetch"
	"mifer/internal/ai/tools/websearch"
	"mifer/pkg/agentconfig"
	"mifer/pkg/logger"
	"mifer/pkg/mcp"
	"mifer/pkg/skill"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// buildCustomAgent 根据用户配置创建 ChatModelAgent。
// 单个工具创建失败时跳过并记录日志，不阻塞整个 Agent 创建；
// 所有工具都创建失败时返回 error。
func buildCustomAgent(
	c context.Context,
	cfg *agentconfig.CustomAgentConfig,
	chatModel model.BaseChatModel,
	mmModel model.BaseChatModel,
	ragSvc rag.RAGService,
	mcpMgr *mcp.Manager,
) (*adk.ChatModelAgent, error) {
	var allTools []tool.BaseTool

	for _, name := range cfg.Tools {
		// MCP 工具引用：mcp:<server_name>
		if strings.HasPrefix(name, "mcp:") {
			serverName := strings.TrimPrefix(name, "mcp:")
			mcpTools, err := mcpMgr.GetToolsForServer(serverName)
			if err != nil {
				logger.Error("获取MCP工具失败",
					logger.S("agent", cfg.Name),
					logger.S("server", serverName),
					logger.C(err))
				continue
			}
			for _, t := range mcpTools {
				allTools = append(allTools, t)
			}
			continue
		}

		// 内置工具：按名创建
		t, err := createTool(name, cfg.BaseDir, mmModel, ragSvc)
		if err != nil {
			logger.Error("创建工具失败",
				logger.S("agent", cfg.Name),
				logger.S("tool", name),
				logger.C(err))
			continue
		}
		allTools = append(allTools, t)
	}

	if len(allTools) == 0 {
		return nil, fmt.Errorf("Agent [%s] 没有成功创建任何工具", cfg.Name)
	}

	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:        cfg.Name,
		Description: cfg.Description,
		Instruction: cfg.Instruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true,
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               allTools,
				ToolCallMiddlewares: []compose.ToolMiddleware{confirmMiddleware},
			},
		},
		MaxIterations: cfg.GetMaxIterations(),
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Agent [%s] 失败: %w", cfg.Name, err)
	}

	return agent, nil
}

// createTool 根据工具名创建对应的 tool.BaseTool，注入外部依赖
func createTool(
	name string,
	baseDir string,
	mmModel model.BaseChatModel,
	ragSvc rag.RAGService,
) (tool.BaseTool, error) {
	switch name {
	case "file_reader":
		return filereader.New()
	case "file_writer":
		if baseDir != "" {
			return filewriter.New(baseDir)
		}
		return filewriter.New()
	case "file_creator":
		if baseDir != "" {
			return filecreator.New(baseDir)
		}
		return filecreator.New()
	case "file_viewer":
		return fileviewer.New(mmModel)
	case "image_generator":
		return imagegenerator.New(mmModel)
	case "command_executor":
		return commandexecutor.New()
	case "knowledge_search":
		if ragSvc == nil {
			return nil, fmt.Errorf("RAG 服务未初始化，无法创建 knowledge_search 工具")
		}
		return knowledgesearch.New(ragSvc)
	case "knowledge_store":
		if ragSvc == nil {
			return nil, fmt.Errorf("RAG 服务未初始化，无法创建 knowledge_store 工具")
		}
		return knowledgestore.New(ragSvc)
	case "web_search":
		return websearch.New()
	case "web_fetch":
		return webfetch.New()
	default:
		return nil, fmt.Errorf("未知工具: %s", name)
	}
}

// buildAllCustomAgents 加载并构建所有自定义 Agent。
// 返回 orchestrator 模式的 Agent 列表（供 deep.New 的 SubAgents 使用），
// 同时将所有 Agent（含 standalone）注册到 AgentHub。
func buildAllCustomAgents(
	c context.Context,
	modelName func(name string) model.BaseChatModel,
	mmModel model.BaseChatModel,
	ragSvc rag.RAGService,
	mcpMgr *mcp.Manager,
	agentHub *skill.AgentHub,
) []adk.Agent {
	configs, err := agentconfig.LoadAgents()
	if err != nil {
		logger.Error("加载自定义Agent配置失败", logger.C(err))
		return nil
	}
	if len(configs) == 0 {
		return nil
	}

	var orchestratorAgents []adk.Agent

	for _, cfg := range configs {
		chatModel := modelName(cfg.GetModel())

		agent, err := buildCustomAgent(c, cfg, chatModel, mmModel, ragSvc, mcpMgr)
		if err != nil {
			logger.Error("构建自定义Agent失败", logger.S("name", cfg.Name), logger.C(err))
			continue
		}

		// 全部注册到 AgentHub（供 skill fork 和 @AgentName 路由使用）
		agentHub.Register(cfg.Name, agent)

		// orchestrator 模式：加入编排器子 Agent 列表
		if cfg.IsOrchestrator() {
			orchestratorAgents = append(orchestratorAgents, agent)
		}

		logger.Info("自定义Agent已加载",
			logger.S("name", cfg.Name),
			logger.S("integration", cfg.Integration),
			logger.I("tools", len(cfg.Tools)))
	}

	return orchestratorAgents
}
