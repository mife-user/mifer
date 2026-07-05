package prompt

import (
	"github.com/cloudwego/eino/schema"
)

// sanitizeMessages 消毒消息序列：移除孤立 tool 消息（没有前驱 assistant+tool_calls 的 tool 消息）。
//
// 当 Deep Agent 多层级事件交织保存或上下文压缩降级时，记忆可能产生孤立 tool 消息，
// 直接发给 LLM API 会报 400 错误。该函数确保 tool 消息始终成对出现。
func sanitizeMessages(msgs []*schema.Message) []*schema.Message {
	filtered := make([]*schema.Message, 0, len(msgs))
	hasPendingToolCalls := false // 上一条 assistant 消息是否包含 tool_calls

	for _, msg := range msgs {
		switch {
		case msg.Role == schema.Assistant && len(msg.ToolCalls) > 0:
			// assistant + tool_calls：标记有待处理的工具调用，保留
			hasPendingToolCalls = true
			filtered = append(filtered, msg)

		case msg.Role == schema.Tool:
			if hasPendingToolCalls {
				// 有对应的 tool_calls，保留
				filtered = append(filtered, msg)
			} else {
				// 孤立 tool 消息，丢弃并记录
				// 注意：不在此处记录日志，避免循环依赖或过度日志
			}

		case msg.Role == schema.Assistant && len(msg.ToolCalls) == 0:
			// 纯文本 assistant 消息：结束当前的 tool_calls 待处理状态
			// 注意：tool_calls 的结束由完整的一轮 [assistant+tool_calls, tool..., tool] 标记，
			// 纯文本 assistant 可能出现在 tool 消息之后（最终回复），也可能独立出现。
			// 重置标记，以便后续工具调用重新开始配对
			hasPendingToolCalls = false
			filtered = append(filtered, msg)

		default:
			// user / system 等：不参与工具配对，保留
			filtered = append(filtered, msg)
		}
	}

	return filtered
}
