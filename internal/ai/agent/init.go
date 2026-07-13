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
)

// initInfra 初始化所有基础设施服务，结果直接写入 Humen 对应字段。
// 同时设置包级 confirmMiddleware，供 Agent 配置复用。
func (h *Humen) initInfra(c context.Context) {
	// RAG 懒加载服务，无网络调用，即时返回
	h.ragSvc = rag.NewLazyService(c)

	// MCP 连接管理器，启动配置中已启用的 Server
	h.MCPManager = mcp.NewManager(conf.GetConfig().Mcp.Servers)

	// 技能管理器
	skillMgr, err := skill.NewManager(conf.GetConfig().Skill)
	if err != nil {
		logger.Warn("技能管理器初始化失败，技能功能不可用", logger.C(err))
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
}

// Init 初始化 Humen：LLM 注册中心 → 基础设施 → Agent/Graph → 记忆与提示词。
func Init(c context.Context) (*Humen, error) {
	reg, err := llm.InitRegistry(c)
	if err != nil {
		logger.Error("初始化LLM注册中心失败", logger.C(err))
		return nil, err
	}
	mmModel := reg.Get("multi_modal")

	h := &Humen{Registry: reg}
	h.initInfra(c)

	if reg.IsReady() {
		subagents, customInfos, err := h.createCustomAgents(c, reg, mmModel)
		if err != nil {
			return nil, err
		}
		h.AgentInfos = append(h.AgentInfos, customInfos...)

		miferAgent, miferInfo, err := h.createMiferAgent(c, reg, mmModel, subagents)
		if err != nil {
			return nil, err
		}
		h.Agents.Mifer = miferAgent
		h.AgentInfos = append(h.AgentInfos, miferInfo)

		if qa, qi := h.createQQAgent(c, reg); qi.Name != "" {
			h.Agents.QQ = qa
			h.AgentInfos = append(h.AgentInfos, qi)
		}

		if pa, pi := h.createPlanAgent(c, reg, mmModel); pi.Name != "" {
			h.Agents.Plan = pa
			h.AgentInfos = append(h.AgentInfos, pi)
			h.Graphs.Plan = newPlanGraph(c, pa, h.ConfirmStore)
		}

		h.Graphs.Habit = createHabitGraph(c, reg.Get("haiku"))
	} else {
		logger.Warn("default后端不可用，跳过Agent初始化，AI对话功能需配置api_key后通过/config重载启用")
		h.AgentInfos = append(h.AgentInfos, AgentInfo{Name: "Mifer", ModelBackend: "default", Description: "Mifer 智能助手（未配置api_key，暂不可用）"})
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

	h.Prompt = prompt.New(mem)
	return h, nil
}
