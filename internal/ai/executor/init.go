package executor

import (
	"context"
	"mifer/internal/ai/agent"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
)

type Executor struct {
	Runner *adk.Runner
	Humen  *agent.Humen
	Token  *TokenUsage // token 累计用量统计
}

func Init(c context.Context, config *conf.Config) (*Executor, error) {

	ag, err := agent.Init(c, config)
	if err != nil {
		logger.Error("初始化agent失败", logger.C(err))
		return nil, err
	}

	runner := adk.NewRunner(c, adk.RunnerConfig{
		Agent:           ag.Agent,
		EnableStreaming: true,
	})
	return &Executor{Runner: runner, Humen: ag, Token: &TokenUsage{}}, nil

}
