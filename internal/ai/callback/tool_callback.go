// Package callback 提供基于 Eino callback 机制的工具调用通知。
// 通过全局 callback handler 在 Tool 组件执行的 OnStart/OnEnd/OnError 时机
// 直接捕获工具调用信息，无需在 executor 的事件迭代器中后处理事件流。
package callback

import (
	"context"
	"encoding/json"

	"mifer/pkg/logger"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/tool"
)

// WithExecutorCallback 将 executor 回调函数注入 context，供 Eino callback handler 使用。
func WithExecutorCallback(ctx context.Context, cb executorCallback) context.Context {
	return context.WithValue(ctx, ctxKey{}, cb)
}

// getExecutorCallback 从 context 中取出 executor 回调，若不存在则返回 nil。
func getExecutorCallback(ctx context.Context) executorCallback {
	if cb, ok := ctx.Value(ctxKey{}).(executorCallback); ok {
		return cb
	}
	return nil
}

// ToolCallbackHandler 是全局注册的 Eino Tool 组件回调处理器。
// 它捕获所有 Tool 调用，通过 context 中注入的 ExecutorCallback 将事件反馈到上层。
var ToolCallbackHandler = callbacks.NewHandlerBuilder().
	OnStartFn(onToolStart).
	OnEndFn(onToolEnd).
	OnErrorFn(onToolError).
	Build()

// onToolStart Tool 组件开始执行时触发。
// 从 CallbackInput 中提取工具名和参数 JSON，发送 tool_start 事件。
func onToolStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil || info.Component != components.ComponentOfTool {
		return ctx
	}

	cb := getExecutorCallback(ctx)
	if cb == nil {
		return ctx
	}

	ti := tool.ConvCallbackInput(input)
	if ti == nil {
		return ctx
	}

	// 与 executor 原有格式保持一致：工具名\x00参数JSON
	if err := cb("tool_start", info.Name+"\x00"+ti.ArgumentsInJSON); err != nil {
		logger.Debug("tool_start 回调失败", logger.S("tool", info.Name), logger.C(err))
	}
	return ctx
}

// onToolEnd Tool 组件执行完成时触发，发送 tool_end 事件，
// 并检查返回值中是否包含 error 字段。
func onToolEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil || info.Component != components.ComponentOfTool {
		return ctx
	}

	cb := getExecutorCallback(ctx)
	if cb == nil {
		return ctx
	}

	// 发送 tool_end
	if err := cb("tool_end", info.Name); err != nil {
		logger.Debug("tool_end 回调失败", logger.S("tool", info.Name), logger.C(err))
	}

	// 检查工具返回值中是否包含 error 字段
	// 工具可能执行成功但返回错误信息（如文件不存在），这种情况 Go error 为 nil，
	// 需要从返回值 JSON 中提取 error 字段
	to := tool.ConvCallbackOutput(output)
	if to != nil && to.Response != "" {
		if errMsg := extractToolError(to.Response); errMsg != "" {
			_ = cb("tool_error", info.Name+"\x00"+errMsg)
		}
	}
	return ctx
}

// onToolError Tool 组件执行出错时触发，发送 tool_error 事件。
func onToolError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil || info.Component != components.ComponentOfTool {
		return ctx
	}

	cb := getExecutorCallback(ctx)
	if cb == nil {
		return ctx
	}

	if cbErr := cb("tool_error", info.Name+"\x00"+err.Error()); cbErr != nil {
		logger.Debug("tool_error 回调失败", logger.S("tool", info.Name), logger.C(cbErr))
	}
	return ctx
}

// extractToolError 从工具返回的 JSON 结果中提取 error 字段的消息。
// 若返回空字符串表示没有错误。
func extractToolError(content string) string {
	var result struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ""
	}
	return result.Error
}
