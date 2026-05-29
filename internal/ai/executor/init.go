package executor

import (
	"context"

	aicallback "mifer/internal/ai/callback"
	"mifer/internal/ai/agent"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
)

type Executor struct {
	Runner *adk.Runner
	Humen  *agent.Humen
	Token  *TokenUsage // token 累计用量统计
}

func Init(c context.Context) (*Executor, error) {
	// 注册全局 Tool 回调处理器，捕获所有 Tool 组件的 OnStart/OnEnd/OnError
	callbacks.AppendGlobalHandlers(aicallback.ToolCallbackHandler)

	ag, err := agent.Init(c)
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
