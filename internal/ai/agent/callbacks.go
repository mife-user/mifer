package agent

import (
	"context"
	"mifer/pkg/logger"
	"time"

	"github.com/cloudwego/eino/adk"
)

// observeHandler 模型调用可观测性：记录每次模型调用的耗时和 token 消耗
//
// 嵌入 BaseChatModelAgentMiddleware，仅覆盖 BeforeModel/AfterModel 两个 hook。
// 经由 deep.Config.Handlers 注入，作用于 DeepAgent 自身的模型调用。
type observeHandler struct {
	*adk.BaseChatModelAgentMiddleware
}

type ctxModelStartKey struct{}

// BeforeModelRewriteState 模型调用前记录开始时间
func (h *observeHandler) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	return context.WithValue(ctx, ctxModelStartKey{}, time.Now()), state, nil
}

// AfterModelRewriteState 模型调用后计算耗时，提取 token 信息
func (h *observeHandler) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	start, _ := ctx.Value(ctxModelStartKey{}).(time.Time)
	if start.IsZero() {
		return ctx, state, nil
	}
	elapsed := time.Since(start)

	// 从最后一条消息提取 token 用量
	var tokens int
	if len(state.Messages) > 0 {
		lastMsg := state.Messages[len(state.Messages)-1]
		if lastMsg != nil && lastMsg.ResponseMeta != nil && lastMsg.ResponseMeta.Usage != nil {
			tokens = lastMsg.ResponseMeta.Usage.TotalTokens
		}
	}

	logger.Debug("模型调用完成",
		logger.S("elapsed", elapsed.String()),
		logger.I("tokens", tokens),
	)
	return ctx, state, nil
}
