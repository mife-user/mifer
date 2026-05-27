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
	Token      *TokenUsage    // token 累计用量统计
	ConfirmBus *tools.ConfirmBus // 工具调用确认总线
}

func Init(c context.Context) (*Executor, error) {
	// 创建确认总线，加载持久白名单
	confirmBus := tools.NewConfirmBus(conf.GetConfig().Cli.Tui.AllowTools)

	ag, err := agent.Init(c, confirmBus)
	if err != nil {
		logger.Error("初始化agent失败", logger.C(err))
		return nil, err
	}

	runner := adk.NewRunner(c, adk.RunnerConfig{
		Agent:           ag.Agent,
		EnableStreaming: true,
	})
	return &Executor{Runner: runner, Humen: ag, Token: &TokenUsage{}, ConfirmBus: confirmBus}, nil

}
