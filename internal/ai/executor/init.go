package executor

import (
	"context"
	"mifer/internal/ai/agent"
	"mifer/pkg/conf"

	"github.com/cloudwego/eino/adk"
)

type Executor struct {
	Runner *adk.Runner
	Humen  *agent.Humen
}

func NewExecutor() *Executor {
	return &Executor{}
}

func Init(c context.Context, config *conf.Config) (*Executor, error) {

	agent, err := agent.Init(c, config)
	if err != nil {
		return nil, err
	}

	runner := adk.NewRunner(c, adk.RunnerConfig{Agent: *agent.Agent})
	return &Executor{Runner: runner, Humen: agent}, nil

}
