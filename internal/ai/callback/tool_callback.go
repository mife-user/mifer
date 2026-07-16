// Package callback 提供基于 Eino callback 机制的工具调用通知。
// 通过构建 per-invocation callback handler，在 Tool 组件执行的 OnStart/OnEnd/OnError
// 时机直接捕获工具调用信息，无需在 executor 的事件迭代器中后处理事件流。
package callback

import (
	"context"
	"encoding/json"

	"mifer/pkg/logger"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/tool"
)

// NewHandler 构建一个 Eino callback handler，将所有 Tool 调用事件通过 cb 发送到上层。
// cb 接收 event（"tool_start" / "tool_end" / "tool_error"）和对应的 content。
// 每次 Run 时通过 adk.WithCallbacks() 注入，替代全局注册。
func NewHandler(cb func(event, content string) error) callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(newOnStart(cb)).
		OnEndFn(newOnEnd(cb)).
		OnErrorFn(newOnError(cb)).
		Build()
}

// newOnStart Tool 组件开始执行时触发。
// 从 CallbackInput 中提取工具名和参数 JSON，发送 tool_start 事件。
func newOnStart(cb func(event, content string) error) func(context.Context, *callbacks.RunInfo, callbacks.CallbackInput) context.Context {
	return func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		if info == nil || info.Component != components.ComponentOfTool {
			return ctx
		}

		ti := tool.ConvCallbackInput(input)
		if ti == nil {
			return ctx
		}

		// 与 executor 原有格式保持一致：工具名\x00参数JSON
		if err := cb("tool_start", info.Name+"\x00"+ti.ArgumentsInJSON); err != nil {
			logger.Debug(ctx, "tool_start 回调失败", logger.S("tool", info.Name), logger.C(err))
		}
		return ctx
	}
}

// newOnEnd Tool 组件执行完成时触发，发送 tool_end 事件，
// 并检查返回值中是否包含 error 字段。
func newOnEnd(cb func(event, content string) error) func(context.Context, *callbacks.RunInfo, callbacks.CallbackOutput) context.Context {
	return func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		if info == nil || info.Component != components.ComponentOfTool {
			return ctx
		}

		// 发送 tool_end
		if err := cb("tool_end", info.Name); err != nil {
			logger.Debug(ctx, "tool_end 回调失败", logger.S("tool", info.Name), logger.C(err))
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
}

// newOnError Tool 组件执行出错时触发，发送 tool_error 事件。
func newOnError(cb func(event, content string) error) func(context.Context, *callbacks.RunInfo, error) context.Context {
	return func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
		if info == nil || info.Component != components.ComponentOfTool {
			return ctx
		}

		if cbErr := cb("tool_error", info.Name+"\x00"+err.Error()); cbErr != nil {
			logger.Debug(ctx, "tool_error 回调失败", logger.S("tool", info.Name), logger.C(cbErr))
		}
		return ctx
	}
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
