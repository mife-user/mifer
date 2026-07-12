package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// confirmMiddleware 包级工具确认中间件，由 Init() 设置后供 Mifer 编排器和自定义 Agent 使用。
var confirmMiddleware compose.ToolMiddleware

type Humen struct {
	Agent        adk.Agent
	QQAgent      adk.Agent // QQ 通道专用 Agent（无工具，仅文本对话），nil 表示未启用
	PlanAgent    adk.Agent // 计划 Agent（只读工具），nil 表示创建失败
	Prompt       *prompt.Prompty
	Registry     *llm.Registry
	MCPManager   *mcp.Manager
	SkillManager *skill.Manager
	ConfirmStore *confirm.Store // 工具确认存储
	AgentInfos   []AgentInfo    // Agent 元数据列表
	HabitGraph   compose.Runnable[[]*schema.Message, string] // 用户习惯总结图
	PlanGraph    compose.Runnable[[]*schema.Message, string] // 计划制定图
}

// qqInstruction QQ 通道 Agent 的系统指令。
const qqInstruction = `你是 Mifer QQ 消息助手。

行为准则：
- 你是 QQ 聊天中的 AI 助手，直接回复用户问题
- 回复简洁自然，与用户消息长度匹配
- 使用与用户相同的语言
- 任务完成后直接结束，不追问
- 你没有文件、命令、搜索等工具，遇到需要这些能力的问题直接说明限制`

// HabitInstruction 用户习惯分析助手的系统指令，供 graph 调用方构建输入消息时使用。
const HabitInstruction = `你是用户习惯分析助手，根据对话内容分析并更新用户画像。

规则：
- 直接以 Markdown 格式输出完整的用户画像内容，不要添加额外说明
- 通过分析每轮对话来推断用户的偏好、习惯、技能和背景
- 如果已有现有画像，基于新对话增量更新，保留之前仍有价值的信息
- 包含但不限于：编程语言偏好、技术栈、工作习惯、常用工具、项目类型、沟通风格
- 内容简洁结构化，使用标题和列表组织
- 不记录敏感信息（密码、密钥、个人身份信息等）`

// miferInstruction Mifer 主 Agent 的系统指令，涵盖所有已注入工具的使用准则。
const miferInstruction = `你是 Mifer 智能助手，运行于 Windows 环境。你直接拥有以下全部工具——无需委派给其他 Agent，所有操作由你独立完成。

可用工具：
- file_reader / file_writer / file_creator / file_viewer / image_generator — 文件读写、创建、查看、图片生成
- command_executor — Windows PowerShell 5.1 命令执行
- knowledge_search / knowledge_store — 知识库检索与文档入库
- web_search / web_fetch — 互联网搜索与网页抓取
- qq_send_message — QQ 消息发送
- skill — 预定义技能调用

Windows 环境：
- 文件路径格式：C:\Users\xxx\file.txt 或 C:/Users/xxx/file.txt，相对路径亦可
- 终端为 PowerShell 5.1：不支持 && / ||，用 ; 分隔命令；环境变量用 $env:NAME
- 常用命令对照：find → Get-ChildItem -Recurse -Filter；grep → Select-String；wc -l → Measure-Object -Line；rm -rf → Remove-Item -Recurse -Force -Confirm:$false；mkdir -p → New-Item -ItemType Directory -Force；touch → New-Item -ItemType File

文件操作铁律（必须遵守）：
1. 写前必读 — 调用 file_writer / file_creator 之前必须先用 file_reader 确认文件当前状态
2. 创建前探测 — 用 file_reader 探测目标路径（不存在会返回错误），确认不存在后再用 file_creator
3. 基于实际修改 — 修改文件必须基于 file_reader 返回的实际内容，禁止凭空猜测

命令执行准则：
- command_executor 仅用于运行程序、构建、安装、测试等命令行操作
- 禁止用 command_executor 读取文件内容（cat/type/Get-Content）——读取文件始终用 file_reader
- 危险命令自动拒绝：递归删除项目外文件、系统关机、权限变更、下载并执行等

知识库准则：
- knowledge_store 存储文档资料（技术文档、会议纪要、需求说明），代码文件不要入库
- 遇到不熟悉的知识点优先用 knowledge_search 检索知识库

工作原则：
1. 先理解用户意图再行动，复杂任务分步执行
2. 涉及实时信息、新闻、最新资料时用 web_search；需要阅读具体网页内容时用 web_fetch
3. 涉及安全操作时审慎评估风险，必要时拒绝并说明原因
4. 工具操作失败时分析错误原因，最多尝试 2 次替代方案；连续 3 次失败后停止并向用户报告
5. 回复使用中文，简洁清晰`

func Init(c context.Context) (*Humen, error) {
	// 初始化 LLM 注册中心
	reg, err := llm.InitRegistry(c)
	if err != nil {
		logger.Error("初始化LLM注册中心失败", logger.C(err))
		return nil, err
	}
	mmModel := reg.Get("multi_modal")
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
	errorMiddleware := tools.NewErrorHandleMiddleware()

	var subagents []adk.Agent
	var agentInfos []AgentInfo
	var agent adk.Agent
	var qqAgent adk.Agent
	var planAgent adk.Agent
	var habitGraph compose.Runnable[[]*schema.Message, string]
	var planGraph compose.Runnable[[]*schema.Message, string]

	// 仅当 default 后端可用时才创建 Agent 和编排器
	// api_key 未配置时跳过，程序运行时通过 /api/admin/status 提示用户
	if defaultReady {
		// 创建自定义 Agent（来自配置文件 agents 段），注入 MCP 工具和技能工具
		for _, agentcfg := range conf.GetConfig().Agents {
			var agentTools []tool.BaseTool
			agentTools, err = tools.NewWithName(agentcfg.Tools, mmModel, ragSvc)
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
						Tools: agentTools,
						ToolCallMiddlewares: []compose.ToolMiddleware{
							errorMiddleware,
							confirmMiddleware,
						},
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
			agentInfos = append(agentInfos, AgentInfo{Name: agentcfg.Name, ModelBackend: agentcfg.Model, Description: agentcfg.Description, Tools: resolveToolNames(c, agentTools)})
		}

		// 构建 Mifer 编排器的工具集 — Mifer 直接持有全部工具，无需委派子 Agent
		orchTools := []tool.BaseTool{skill.NewSkillTool(skillMgr, skillHub)}
		// MCP 工具
		for _, t := range mcpToBaseTools(mcpManager.GetToolsForAgent("Mifer")) {
			orchTools = append(orchTools, t)
		}
		// 网页搜索与抓取
		for _, t := range tools.WebTools() {
			orchTools = append(orchTools, t)
		}
		// QQ 消息发送
		for _, t := range tools.QQTools(func() qq.Sender { return nil }) {
			orchTools = append(orchTools, t)
		}
		// 文件操作（读取、写入、创建、查看、图片生成）
		for _, t := range tools.FileTools(mmModel) {
			orchTools = append(orchTools, t)
		}
		// 终端命令执行
		for _, t := range tools.CommandTools() {
			orchTools = append(orchTools, t)
		}
		// 并行调度（仅在注册了自定义 Agent 时添加）
		if skillHub.HasAny() {
			for _, t := range tools.ParallelDispatch(skillHub) {
				orchTools = append(orchTools, t)
			}
		}
		// 知识库检索与存储
		for _, t := range tools.KnowledgeTools(ragSvc) {
			orchTools = append(orchTools, t)
		}

		// 创建 Mifer 编排器（deep.New），自定义 Agent 作为可选子 Agent 保留
		agent, err = deep.New(c, &deep.Config{
			Name:        "Mifer",
			Description: "Mifer 智能助手，具备文件操作、命令执行、知识库管理、网页搜索和 QQ 消息等完整能力",
			Instruction: miferInstruction,
			ChatModel:   reg.Get("default"),
			ToolsConfig: adk.ToolsConfig{
				EmitInternalEvents: true,
				ToolsNodeConfig: compose.ToolsNodeConfig{
					Tools: orchTools,
					ToolCallMiddlewares: []compose.ToolMiddleware{
						errorMiddleware,
						confirmMiddleware,
					},
				},
			},
			SubAgents:    subagents,
			MaxIteration: 100,
		})
		if err != nil {
			logger.Error("init agent failed", logger.C(err))
			return nil, err
		}
		agentInfos = append(agentInfos, AgentInfo{Name: "Mifer", ModelBackend: "default", Description: "Mifer 智能助手，具备文件操作、命令执行、知识库管理、网页搜索和 QQ 消息等完整能力", Tools: resolveToolNames(c, orchTools)})

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

			// 创建 PlanAgent（只读工具，sonnet 模型，复用中间件栈）
			if pa, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
				Name:        "PlanAgent",
				Description: "计划制定助手，只能查看文件和搜索，不可写入或执行",
				Instruction: PlanInstruction,
				Model:       reg.Get("sonnet"),
				ToolsConfig: adk.ToolsConfig{
					ToolsNodeConfig: compose.ToolsNodeConfig{
						Tools: tools.ReadOnlyTools(mmModel, ragSvc),
						ToolCallMiddlewares: []compose.ToolMiddleware{
							errorMiddleware,
							confirmMiddleware,
						},
					},
				},
				MaxIterations: 20,
			}); err == nil {
				planAgent = pa
				agentInfos = append(agentInfos, AgentInfo{Name: "PlanAgent", ModelBackend: "sonnet", Description: "计划制定助手，只读分析"})
			}

			// 创建 PlanGraph（计划制定图）
			if planAgent != nil {
				planGraph = newPlanGraph(c, planAgent, confirmStore)
			}

		habitGraph = newHabitGraph(c, reg.Get("haiku"))
	} else {
		logger.Warn("default后端不可用，跳过Agent初始化，AI对话功能需配置api_key后通过/config重载启用")
		agentInfos = append(agentInfos, AgentInfo{Name: "Mifer", ModelBackend: "default", Description: "Mifer 智能助手（未配置api_key，暂不可用）"})
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
	return &Humen{Agent: agent, QQAgent: qqAgent, Prompt: prompty, Registry: reg, MCPManager: mcpManager, SkillManager: skillMgr, ConfirmStore: confirmStore, AgentInfos: agentInfos, HabitGraph: habitGraph, PlanAgent: planAgent, PlanGraph: planGraph}, nil
}

// newHabitGraph 创建用户习惯总结图：ChatModel(haiku) → Lambda(写入 MIFER.md)
// 图结构：START → habit_chat → habit_writer → END
// 返回 nil 表示图创建失败（仅记录日志，不中断 Init 流程）
func newHabitGraph(ctx context.Context, chatModel model.BaseChatModel) compose.Runnable[[]*schema.Message, string] {
	g := compose.NewGraph[[]*schema.Message, string]()

	if err := g.AddChatModelNode("habit_chat", chatModel); err != nil {
		logger.Warn("创建习惯总结图 ChatModel 节点失败", logger.C(err))
		return nil
	}
	if err := g.AddLambdaNode("habit_writer", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (string, error) {
			miferPath := filepath.Join(conf.GetConfig().Path.CfgPath, "MIFER.md")
			if err := os.WriteFile(miferPath, []byte(msg.Content), 0644); err != nil {
				return "", fmt.Errorf("写入 MIFER.md 失败: %w", err)
			}
			logger.Debug("用户画像已更新", logger.I("len", len(msg.Content)))
			return msg.Content, nil
		},
	)); err != nil {
		logger.Warn("创建习惯总结图 Lambda 节点失败", logger.C(err))
		return nil
	}
	if err := g.AddEdge(compose.START, "habit_chat"); err != nil {
		logger.Warn("创建习惯总结图边失败", logger.C(err))
		return nil
	}
	if err := g.AddEdge("habit_chat", "habit_writer"); err != nil {
		logger.Warn("创建习惯总结图边失败", logger.C(err))
		return nil
	}
	if err := g.AddEdge("habit_writer", compose.END); err != nil {
		logger.Warn("创建习惯总结图边失败", logger.C(err))
		return nil
	}

	compiled, err := g.Compile(ctx)
	if err != nil {
		logger.Warn("编译习惯总结图失败", logger.C(err))
		return nil
	}
	return compiled
}

// mcpToBaseTools 将 []tool.InvokableTool 转为 []tool.BaseTool
func mcpToBaseTools(invokable []tool.InvokableTool) []tool.BaseTool {
	var result []tool.BaseTool
	for _, t := range invokable {
		result = append(result, t)
	}
	return result
}
