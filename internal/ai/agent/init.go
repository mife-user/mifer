package agent

import (
	"context"
	"time"

	"mifer/internal/ai/confirm"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/memory"
	"mifer/internal/ai/prompt"
	"mifer/internal/ai/rag"
	"mifer/internal/ai/tools"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"mifer/pkg/mcp"
	"mifer/pkg/skill"
	"mifer/qq"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// confirmMiddleware 包级工具确认中间件，由 Init() 设置后供各 Agent 构造函数使用。
var confirmMiddleware compose.ToolMiddleware

type Humen struct {
	Agent        adk.Agent
	QQAgent      adk.Agent // QQ 通道专用 Agent（无工具，仅文本对话），nil 表示未启用
	Prompt       *prompt.Prompty
	Registry     *llm.Registry
	MCPManager   *mcp.Manager
	SkillManager *skill.Manager
	ConfirmStore *confirm.Store // 工具确认存储
	AgentInfos   []AgentInfo    // Agent 元数据列表
}

// qqInstruction QQ 通道 Agent 的系统指令。
const qqInstruction = `你是 Mifer QQ 消息助手。

行为准则：
- 你是 QQ 聊天中的 AI 助手，直接回复用户问题
- 回复简洁自然，与用户消息长度匹配
- 使用与用户相同的语言
- 任务完成后直接结束，不追问
- 你没有文件、命令、搜索等工具，遇到需要这些能力的问题直接说明限制`

func Init(c context.Context) (*Humen, error) {
	// 初始化LLM注册中心
	reg, err := llm.InitRegistry(c)
	if err != nil {
		logger.Error("初始化LLM注册中心失败", logger.C(err))
		return nil, err
	}

	// 检查 default 后端是否可用（api_key 未配置时不可用）
	defaultReady := reg.IsReady()

	// RAG 懒加载服务，无网络调用，即时返回，Qdrant 连接推迟到首次工具调用
	ragSvc := rag.NewLazyService(c)

	// 初始化 MCP 连接管理器，启动配置中已启用的 Server
	mcpManager := mcp.NewManager(conf.GetConfig().Mcp.Servers)

	// 初始化技能管理器
	skillMgr, err := skill.NewManager(conf.GetConfig().Skill)
	if err != nil {
		logger.Warn("技能管理器初始化失败，技能功能不可用", logger.C(err))
	}
	skillHub := skill.NewAgentHub()

	// 初始化工具确认存储与中间件
	confirmStore := confirm.NewStore()
	confirmCfg := conf.GetConfig().Confirm
	confirmMiddleware = confirm.NewConfirmMiddleware(
		confirmStore,
		confirm.NeedConfirm(confirmStore),
		time.Duration(confirmCfg.TimeoutSec)*time.Second,
	)
	var subagents []adk.Agent
	var agentInfos []AgentInfo
	var agent adk.Agent
	var qqAgent adk.Agent

	// 仅当 default 后端可用时才创建子 Agent 和编排器
	// api_key 未配置时跳过，程序运行时通过 /api/admin/status 提示用户
	if defaultReady {
		//初始化图片创建agent（multi_modal — 图片生成）
		imageAgent, err := newImage(c, reg.Get("multi_modal"), reg.Get("multi_modal"), mcpToBaseTools(mcpManager.GetToolsForAgent("MiImager")))
		if err != nil {
			logger.Error("init image agent failed", logger.C(err))
			return nil, err
		}
		skillHub.Register("MiImager", imageAgent)
		subagents = append(subagents, imageAgent)
		imagerInfo := AgentInfo{Name: "MiImager", ModelBackend: "multi_modal", Description: "图片生成专家，负责创建和编辑图片", Tools: resolveToolNames(c, mcpToBaseTools(mcpManager.GetToolsForAgent("MiImager")))}
		imagerInfo.Tools = append(imagerInfo.Tools, "image_generator", "file_viewer")
		agentInfos = append(agentInfos, imagerInfo)
		// 初始化文档摘要agent（sonnet — 均衡）
		summarizerAgent, err := newSummarizer(c, reg.Get("sonnet"), reg.Get("multi_modal"), tools.KnowledgeTools(ragSvc), mcpToBaseTools(mcpManager.GetToolsForAgent("MiSummarizer")))
		if err != nil {
			logger.Error("init summarizer agent failed", logger.C(err))
			return nil, err
		}
		skillHub.Register("MiSummarizer", summarizerAgent)
		subagents = append(subagents, summarizerAgent)
		summarizerAgentInfo := AgentInfo{Name: "MiSummarizer", ModelBackend: "sonnet", Description: "文档摘要与知识库管理专家，读取文档、查看图片、生成摘要并可检索和存储知识库", Tools: resolveToolNames(c, mcpToBaseTools(mcpManager.GetToolsForAgent("MiSummarizer")))}
		summarizerAgentInfo.Tools = append(summarizerAgentInfo.Tools, "file_viewer", "knowledge_search", "knowledge_store")
		agentInfos = append(agentInfos, summarizerAgentInfo)
		for _, agentcfg := range conf.GetConfig().Agents {
			var tool []tool.BaseTool
			tool, err = tools.NewWithName(agentcfg.Tools, reg.Get("multi_modal"), ragSvc)
			if err != nil {
				logger.Error("创建自定义Agent工具失败", logger.S("agent", agentcfg.Name), logger.C(err))
				return nil, err
			}
			extraAgent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
				Name:        agentcfg.Name,
				Description: agentcfg.Description,
				Instruction: agentcfg.Instruction,
				Model:       reg.Get(agentcfg.Model),
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools:               tool,
						ToolCallMiddlewares: []compose.ToolMiddleware{confirmMiddleware},
					},
				},

				MaxIterations: 100,
			})
			if err != nil {
				logger.Error("创建自定义Agent失败", logger.S("name", agentcfg.Name), logger.C(err))
				return nil, err
			}
			subagents = append(subagents, extraAgent)
			skillHub.Register(agentcfg.Name, extraAgent)
			agentInfos = append(agentInfos, AgentInfo{Name: agentcfg.Name, ModelBackend: agentcfg.Model, Description: agentcfg.Description, Tools: resolveToolNames(c, tool)})
		}
		// 构建编排器的工具集
		orchTools := []tool.BaseTool{skill.NewSkillTool(skillMgr, skillHub)}
		for _, t := range mcpToBaseTools(mcpManager.GetToolsForAgent("Mifer")) {
			orchTools = append(orchTools, t)
		}
		for _, t := range tools.WebTools() {
			orchTools = append(orchTools, t)
		}
		for _, t := range tools.QQTools(func() qq.Sender { return nil }) {
			orchTools = append(orchTools, t)
		}
		backend, err := localbk.NewBackend(c, &localbk.Config{})
		if err != nil {
			logger.Error(errorer.ErrInitLocalBackendFailed, logger.C(err))
			return nil, err
		}
		// 初始化编排器agent（default — 调度主脑）
		agent, err = deep.New(c, &deep.Config{
			Name:        "Mifer",
			Description: "智能任务编排器，根据用户请求自动选择最合适的专家Agent处理任务",
			Instruction: miferInstruction,
			ChatModel:   reg.Get("default"),
			ToolsConfig: adk.ToolsConfig{
				EmitInternalEvents: true,
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools:               orchTools,
					ToolCallMiddlewares: []compose.ToolMiddleware{confirmMiddleware},
				},
			},
			SubAgents:      subagents,
			Backend:        backend,
			StreamingShell: backend,
			MaxIteration:   100,
		})
		if err != nil {
			logger.Error("init agent failed", logger.C(err))
			return nil, err
		}
		agentInfos = append(agentInfos, AgentInfo{Name: "Mifer", ModelBackend: "default", Description: "智能任务编排器，根据用户请求自动选择最合适的专家Agent处理任务", Tools: resolveToolNames(c, orchTools)})

		// 创建 QQ 通道专用 Agent（无工具，纯文本对话）
		// 使用 ChatModelAgent 而非 DeepAgent：QQ 消息不需要子 Agent 调度，直接回复即可
		if qa, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
			Name:          "MiQQ",
			Description:   "QQ 消息助手，直接回复用户问题",
			Instruction:   qqInstruction,
			Model:         reg.Get("default"),
			MaxIterations: 1,
		}); err == nil {
			qqAgent = qa
			agentInfos = append(agentInfos, AgentInfo{Name: "MiQQ", ModelBackend: "default", Description: "QQ 通道专用助手，纯文本对话"})
		}
	} else {
		logger.Warn("default后端不可用，跳过Agent初始化，AI对话功能需配置api_key后通过/config重载启用")
		agentInfos = append(agentInfos, AgentInfo{Name: "Mifer", ModelBackend: "default", Description: "智能任务编排器（未配置api_key，暂不可用）"})
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
	return &Humen{Agent: agent, QQAgent: qqAgent, Prompt: prompty, Registry: reg, MCPManager: mcpManager, SkillManager: skillMgr, ConfirmStore: confirmStore, AgentInfos: agentInfos}, nil
}

// mcpToBaseTools 将 []tool.InvokableTool 转为 []tool.BaseTool
func mcpToBaseTools(invokable []tool.InvokableTool) []tool.BaseTool {
	var result []tool.BaseTool
	for _, t := range invokable {
		result = append(result, t)
	}
	return result
}
const miferInstruction = ` 你是Mifer智能助手的主控编排器，运行于Windows环境，负责分析用户请求并调度合适的专家Agent。

你可以调用的专家Agent：
- MiImager：图片查看与生成（基于图片描述生成图片文件、查看图片内容）
- MiSummarizer：文档阅读、摘要总结与知识库管理（支持知识库检索、文档入库、文档和图片查看）

你自身具备以下工具：
- web_search：搜索互联网获取最新信息（基于 SearXNG 元搜索引擎，聚合 Google/Bing/百度等多家结果）
- web_fetch：抓取指定网页URL的文本内容
- skill：调用预定义的技能（支持 inline 和 fork 两种模式）
- qq_send_message：发送QQ消息（仅QQ通道可用）

Windows 环境须知：
- 文件路径使用反斜杠（如 C:\Users\xxx\file.txt）或正斜杠均可
- 工作目录为当前项目根目录，所有相对路径基于此目录
- 你自身不具备文件读写工具，所有文件操作必须委派给 MiImager 或 MiSummarizer

Agent 委派原则：
1. 图片生成、图片查看、文件内容查看 → 委派给 MiImager
2. 文档阅读、知识库检索、知识库存储 → 委派给 MiSummarizer
3. 涉及代码文件查看 → 委派给 MiSummarizer（使用 file_viewer）
4. 复杂任务可串联多个Agent协作完成
5. 自定义 Agent 在技能列表中查看，按描述匹配合适的 Agent

工作原则：
1. 先理解用户意图，再选择合适的Agent或工具
2. 涉及实时信息、新闻、最新资料时使用 web_search 搜索
3. 需要阅读具体网页内容时使用 web_fetch 抓取
4. 子Agent连续3次失败后，不要再委派相同任务，向用户报告失败原因并等待指示
5. 回复用户时使用中文，简洁清晰
6. 不需要向用户展示Agent的思考过程或内部调度细节，直接呈现最终结果`