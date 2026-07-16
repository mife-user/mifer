package tools

import (
	"context"
	"fmt"
	"strings"

	"mifer/pkg/logger"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ToolExchangeWriter 工具调用交换记录的写入接口。
// memory.Memory 实现了此接口，将 ToolCall + ToolResult 追加到对话历史。
type ToolExchangeWriter interface {
	AppendToolExchange(assistantWithToolCall, toolResult *schema.Message)
}

// NewPersistenceMiddleware 创建工具持久化中间件。
// 拦截所有通过确认的工具调用，将 ToolCall + ToolResult 通过 writer 写入持久化存储。
// 该中间件应置于 confirmMiddleware 内层，确保只记录已确认的调用。
func NewPersistenceMiddleware(writer ToolExchangeWriter) compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable:         createInvokablePersistence(writer),
		EnhancedInvokable: createEnhancedInvokablePersistence(writer),
	}
}

// createInvokablePersistence 非流式普通工具的持久化处理。
func createInvokablePersistence(writer ToolExchangeWriter) compose.InvokableToolMiddleware {
	return func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
			output, err := next(ctx, input)
			if err != nil {
				return nil, err
			}

			persistToolExchange(writer, input.Name, input.Arguments, input.CallID, output.Result)
			return output, nil
		}
	}
}

// createEnhancedInvokablePersistence 非流式增强工具的持久化处理。
func createEnhancedInvokablePersistence(writer ToolExchangeWriter) compose.EnhancedInvokableToolMiddleware {
	return func(next compose.EnhancedInvokableToolEndpoint) compose.EnhancedInvokableToolEndpoint {
		return func(ctx context.Context, input *compose.ToolInput) (*compose.EnhancedInvokableToolOutput, error) {
			output, err := next(ctx, input)
			if err != nil {
				return nil, err
			}

			resultText := toolResultToText(output.Result)
			persistToolExchange(writer, input.Name, input.Arguments, input.CallID, resultText)
			return output, nil
		}
	}
}

// persistToolExchange 将一次工具调用交互构建为消息对并写入。
func persistToolExchange(writer ToolExchangeWriter, toolName, arguments, callID, resultText string) {
	if writer == nil {
		return
	}

	toolCall := schema.ToolCall{
		ID:   callID,
		Type: "function",
		Function: schema.FunctionCall{
			Name:      toolName,
			Arguments: arguments,
		},
	}
	assistantMsg := schema.AssistantMessage("", []schema.ToolCall{toolCall})

	toolResultMsg := schema.ToolMessage(resultText, toolName)
	toolResultMsg.ToolCallID = callID

	writer.AppendToolExchange(assistantMsg, toolResultMsg)

	logger.Debug(context.Background(), "已持久化工具调用",
		logger.S("tool", toolName),
		logger.S("callID", callID),
		logger.I("resultLen", len(resultText)),
	)
}

// toolResultToText 将增强工具的多模态结果转换为纯文本表示，用于持久化。
func toolResultToText(result *schema.ToolResult) string {
	if result == nil || len(result.Parts) == 0 {
		return ""
	}

	var parts []string
	for _, part := range result.Parts {
		switch part.Type {
		case schema.ToolPartTypeText:
			parts = append(parts, part.Text)
		case schema.ToolPartTypeImage:
			url := ""
			if part.Image.URL != nil {
				url = *part.Image.URL
			}
			parts = append(parts, fmt.Sprintf("[图片: %s]", url))
		case schema.ToolPartTypeAudio:
			parts = append(parts, "[音频]")
		case schema.ToolPartTypeVideo:
			parts = append(parts, "[视频]")
		case schema.ToolPartTypeFile:
			parts = append(parts, "[文件]")
		default:
			parts = append(parts, "[未知类型]")
		}
	}

	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, "\n")
}
