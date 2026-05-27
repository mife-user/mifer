package agent

import (
	"context"
	"mifer/internal/ai/tools"

	"github.com/cloudwego/eino/compose"
)

// makeConfirmMiddleware 创建工具调用确认中间件
//
// 当 bus 为 nil 时返回 nil（无确认模式）。
// 中间件在每次工具执行前调用 ConfirmBus.Confirm() 等待 TUI 确认。
func makeConfirmMiddleware(bus *tools.ConfirmBus) compose.InvokableToolMiddleware {
	if bus == nil {
		return nil
	}
	return func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			if err := bus.Confirm(input.Name, input.Arguments); err != nil {
				return nil, err
			}
			return next(ctx, input)
		}
	}
}
