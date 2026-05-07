package executor

import (
	"context"
	"mifer/internal/ai/agent"
	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
)

type Executor struct {
	Runner *adk.Runner
	Humen  *agent.Humen
}

func NewExecutor() *Executor {
	return &Executor{}
}

func Init(c context.Context, config *conf.Config) (domain.Agent, error) {

	agent, err := agent.Init(c, config)
	if err != nil {
		logger.Error("初始化agent失败", logger.C(err))
		return nil, err
	}

	runner := adk.NewRunner(c, adk.RunnerConfig{
		Agent:           agent.Agent,
		EnableStreaming: false,
	})
	return &Executor{Runner: runner, Humen: agent}, nil

}
