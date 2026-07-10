package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewErrorHandleMiddleware 创建工具错误处理中间件。
// 拦截工具调用链中返回的 Go error，将其转换为 LLM 可读的文字响应，
// 避免 error 传播到 Eino 调用链导致对话中断。
// 该中间件应置于中间件链最外层，以便同时捕获下游中间件（如确认拒绝）产生的错误。
func NewErrorHandleMiddleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable:  createInvokableErrorHandler(),
		Streamable: createStreamableErrorHandler(),
	}
}

// createInvokableErrorHandler 非流式工具错误处理。
func createInvokableErrorHandler() compose.InvokableToolMiddleware {
	return func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			output, err := next(ctx, input)
			if err != nil {
				return &compose.ToolOutput{
					Result: fmt.Sprintf("工具 [%s] 执行出错: %s", input.Name, err.Error()),
				}, nil
			}
			return output, nil
		}
	}
}

// createStreamableErrorHandler 流式工具错误处理。
// 流式工具出错时返回包含错误文本的单元素流，框架消费后 LLM 获得文字错误提示。
func createStreamableErrorHandler() compose.StreamableToolMiddleware {
	return func(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.StreamToolOutput, error) {
			output, err := next(ctx, input)
			if err != nil {
				return &compose.StreamToolOutput{
					Result: schema.StreamReaderFromArray([]string{
						fmt.Sprintf("工具 [%s] 执行出错: %s", input.Name, err.Error()),
					}),
				}, nil
			}
			return output, nil
		}
	}
}
