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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// initInfra 初始化所有基础设施服务，结果直接写入 Humen 对应字段。
// 同时设置包级 confirmMiddleware 和 persistenceMw，供 Agent 配置复用。
func (h *Humen) initInfra(c context.Context) {
	// RAG 懒加载服务，无网络调用，即时返回
	h.ragSvc = rag.NewLazyService(c)

	// MCP 连接管理器，启动配置中已启用的 Server
	h.MCPManager = mcp.NewManager(conf.GetConfig().Mcp.Servers)

	// 技能管理器
	skillMgr, err := skill.NewManager(conf.GetConfig().Skill)
	if err != nil {
		logger.Warn(c, "技能管理器初始化失败，技能功能不可用", logger.C(err))
	}
	h.SkillManager = skillMgr
	h.skillHub = skill.NewAgentHub()

	// 工具确认存储与中间件
	h.ConfirmStore = confirm.NewStore()
	confirmCfg := conf.GetConfig().Confirm
	confirmMiddleware = confirm.NewConfirmMiddleware(
		h.ConfirmStore,
		confirm.NeedConfirm(h.ConfirmStore),
		time.Duration(confirmCfg.TimeoutSec)*time.Second,
	)
	h.errorMw = tools.NewErrorHandleMiddleware()

	// 工具持久化中间件
	h.persistenceMw = newPersistenceMiddleware(h.Prompt.Memory)
}

// getBackendModel 根据 Agent 名称从配置中获取对应的模型实例。
// 回退链：config.Ai.AgentBackends[agentName] → 第一个注册后端 → nil
func getBackendModel(reg *llm.Registry, agentName string) model.BaseChatModel {
	cfg := conf.GetConfig()
	backendName, ok := cfg.Ai.AgentBackends[agentName]
	if !ok || backendName == "" {
		backendName = reg.FirstKey()
	}
	if backendName == "" {
		return nil
	}
	return reg.Get(backendName)
}

// Init 初始化 Humen：LLM 注册中心 → Memory + Prompt → 基础设施 → Agent/Graph。
func Init(c context.Context) (*Humen, error) {
	reg, err := llm.InitRegistry(c)
	if err != nil {
		logger.Error(c, "初始化LLM注册中心失败", logger.C(err))
		return nil, err
	}

	h := &Humen{Registry: reg}

	// Memory 提前创建，供 persistenceMw 初始化时引用
	id, ok := c.Value("id").(string)
	if !ok {
		logger.Error(c, "id is not string")
		return nil, errorer.New(errorer.ErrIDNotString)
	}
	mem, err := memory.Init(id)
	if err != nil {
		logger.Error(c, "init memory failed", logger.C(err))
		return nil, err
	}
	h.Prompt = prompt.New(mem)

	// 初始化基础设施（含 persistenceMw，可通过 h.Prompt.Memory 访问 Memory）
	h.initInfra(c)

	if reg.IsReady() {
		subagents, customInfos, err := h.createCustomAgents(c, reg)
		if err != nil {
			return nil, err
		}
		h.AgentInfos = append(h.AgentInfos, customInfos...)

		miferAgent, miferInfo, err := h.createMiferAgent(c, reg, subagents)
		if err != nil {
			return nil, err
		}
		if miferInfo.Name != "" {
			h.Agents.Mifer = miferAgent
			h.AgentInfos = append(h.AgentInfos, miferInfo)
		}

		if qa, qi := h.createQQAgent(c, reg); qi.Name != "" {
			h.Agents.QQ = qa
			h.AgentInfos = append(h.AgentInfos, qi)
		}

		if pa, pi := h.createPlanAgent(c, reg); pi.Name != "" {
			h.Agents.Plan = pa
			h.AgentInfos = append(h.AgentInfos, pi)
			h.Graphs.Plan = newPlanGraph(c, pa, h.ConfirmStore)
		}

		habitModel := getBackendModel(reg, "habit_summarizer")
		if habitModel != nil {
			h.Graphs.Habit = createHabitGraph(c, habitModel)
		}
	} else {
		logger.Warn(c, "没有可用的模型后端，跳过Agent初始化，请配置 ai.backends 后通过 /config 重载启用")
		h.AgentInfos = append(h.AgentInfos, AgentInfo{Name: "Mifer", ModelBackend: "", Description: "Mifer 智能助手（未配置可用后端，暂不可用）"})
	}

	return h, nil
}

// newPersistenceMiddleware 创建工具持久化中间件实例。
func newPersistenceMiddleware(mem *memory.Memory) compose.ToolMiddleware {
	return tools.NewPersistenceMiddleware(mem)
}
