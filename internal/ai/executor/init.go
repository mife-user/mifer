package executor

import (
	"context"
	"mifer/internal/ai/agent"
	"mifer/internal/ai/tools"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
)

type Executor struct {
	Runner     *adk.Runner
	Humen      *agent.Humen
	Token      *TokenUsage        // token 累计用量统计
	ConfirmBus *tools.ConfirmBus  // 工具调用确认总线
	PlanGraph  *planRunner        // 计划 Graph 运行器
}

func Init(c context.Context) (*Executor, error) {
	// 创建确认总线，加载持久白名单
	cfg := conf.GetConfig().Cli.Tui
	confirmBus := tools.NewConfirmBus(cfg.AllowTools, convertAllowToolRules(cfg.AllowToolRules))

	ag, err := agent.Init(c, confirmBus)
	if err != nil {
		logger.Error("初始化agent失败", logger.C(err))
		return nil, err
	}

	runner := adk.NewRunner(c, adk.RunnerConfig{
		Agent:           ag.Agent,
		EnableStreaming: true,
	})
	// 创建计划 Graph 运行器
	planGraph, err := newPlanRunner(c, ag.Registry.Get("default"), confirmBus, conf.GetConfig().Path.Workdir)
	if err != nil {
		logger.Error("初始化计划 Graph 失败", logger.C(err))
		return nil, err
	}

	return &Executor{Runner: runner, Humen: ag, Token: &TokenUsage{}, ConfirmBus: confirmBus, PlanGraph: planGraph}, nil

}

// convertAllowToolRules 将配置中的 AllowToolRule 转换为 tools.AllowToolRule
func convertAllowToolRules(rules []conf.AllowToolRule) []tools.AllowToolRule {
	result := make([]tools.AllowToolRule, len(rules))
	for i, r := range rules {
		result[i] = tools.AllowToolRule{
			Tool:        r.Tool,
			ArgsPattern: r.ArgsPattern,
		}
	}
	return result
}
