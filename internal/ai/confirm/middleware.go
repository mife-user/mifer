package confirm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/compose"
	"github.com/google/uuid"
)

// WithCallback 将 executor 回调函数注入 context，供中间件发送 SSE 事件。
func WithCallback(ctx context.Context, cb ExecutorCallback) context.Context {
	return context.WithValue(ctx, ctxKey{}, cb)
}

// getCallback 从 context 中取出 executor 回调。
func getCallback(ctx context.Context) ExecutorCallback {
	if cb, ok := ctx.Value(ctxKey{}).(ExecutorCallback); ok {
		return cb
	}
	return nil
}

// WithSessionID 将会话 ID 注入 context。
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, id)
}

// getSessionID 从 context 中取出会话 ID。
func getSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(sessionIDKey{}).(string); ok {
		return id
	}
	return ""
}

// NewConfirmMiddleware 创建 Eino ToolMiddleware，在工具调用前拦截并等待确认。
// needConfirmFn 接收 toolName、arguments、sessionID，返回是否需要确认。
// timeout 为等待用户确认的最大时长。
func NewConfirmMiddleware(store *Store,
	needConfirmFn func(toolName, arguments, sessionID string) bool,
	timeout time.Duration) compose.ToolMiddleware {

	return compose.ToolMiddleware{
		Invokable:  createInvokableMiddleware(store, needConfirmFn, timeout),
		Streamable: createStreamableMiddleware(store, needConfirmFn, timeout),
	}
}

// createInvokableMiddleware 创建非流式工具调用的确认中间件。
func createInvokableMiddleware(store *Store,
	needConfirmFn func(string, string, string) bool,
	timeout time.Duration) compose.InvokableToolMiddleware {

	return func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			sessionID := getSessionID(ctx)
			if !needConfirmFn(input.Name, input.Arguments, sessionID) {
				return next(ctx, input)
			}

			cb := getCallback(ctx)
			result, err := awaitConfirmation(ctx, cb, store, input.Name, input.Arguments, input.CallID, sessionID, timeout)
			if err != nil {
				return nil, err
			}

			// 记录 "allow" 到 session 白名单
			if result.Action == "allow" {
				store.AllowTool(sessionID, input.Name)
			}

			return next(ctx, input)
		}
	}
}

// createStreamableMiddleware 创建流式工具调用的确认中间件。
func createStreamableMiddleware(store *Store,
	needConfirmFn func(string, string, string) bool,
	timeout time.Duration) compose.StreamableToolMiddleware {

	return func(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.StreamToolOutput, error) {
			sessionID := getSessionID(ctx)
			if !needConfirmFn(input.Name, input.Arguments, sessionID) {
				return next(ctx, input)
			}

			cb := getCallback(ctx)
			result, err := awaitConfirmation(ctx, cb, store, input.Name, input.Arguments, input.CallID, sessionID, timeout)
			if err != nil {
				return nil, err
			}

			if result.Action == "allow" {
				store.AllowTool(sessionID, input.Name)
			}

			return next(ctx, input)
		}
	}
}

// sendConfirmEvent 发送 tool_confirm SSE 事件，失败仅记录日志。
func sendConfirmEvent(cb ExecutorCallback, id, toolName, arguments string) {
	eventData, err := json.Marshal(ConfirmEvent{
		ID: id, ToolName: toolName, Arguments: arguments,
	})
	if err != nil {
		logger.Error("序列化 tool_confirm 事件失败", logger.C(err))
		return
	}

	logger.Info("发送 tool_confirm SSE 事件",
		logger.S("id", id), logger.S("tool", toolName),
		logger.I("argsLen", len(arguments)))

	if sendErr := cb("tool_confirm", string(eventData)); sendErr != nil {
		logger.Error("发送 tool_confirm SSE 事件失败",
			logger.S("tool", toolName), logger.C(sendErr))
	}
}

// awaitConfirmation 生成确认项、发送 SSE 事件、阻塞等待确认结果。
func awaitConfirmation(ctx context.Context, cb ExecutorCallback, store *Store,
	toolName, arguments, callID, sessionID string, timeout time.Duration) (ConfirmResult, error) {

	id := uuid.New().String()
	entry := &PendingEntry{
		ID:        id,
		ToolName:  toolName,
		Arguments: arguments,
		CallID:    callID,
		ResultCh:  make(chan ConfirmResult, 1),
		CreatedAt: time.Now(),
		SessionID: sessionID,
	}
	store.Add(entry)
	defer store.Remove(id)

	if cb != nil {
		sendConfirmEvent(cb, id, toolName, arguments)
	} else {
		logger.Warn("tool_confirm 无法发送：callback 为 nil",
			logger.S("tool", toolName), logger.S("id", id))
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case r := <-entry.ResultCh:
		if !r.Approved {
			return r, fmt.Errorf("工具调用被用户拒绝: %s", r.Action)
		}
		return r, nil
	case <-timer.C:
		return ConfirmResult{Action: "timeout"}, errorer.New(errorer.ErrConfirmTimeout)
	case <-ctx.Done():
		return ConfirmResult{Action: "canceled"}, errorer.New(errorer.ErrConfirmDone)
	}
}
