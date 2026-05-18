package executor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	e.Humen.Memory.AppendUser(req.Content)

	iter := e.Runner.Run(c, e.Humen.Memory.Messages)

	lastMsg := &strings.Builder{}
	eventCount := 0
	var currentAgent string // 跟踪当前执行的Agent，用于检测切换
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++
		if event.Err != nil {
			logger.Error("AI事件错误", logger.C(event.Err))
			return event.Err
		}

		// 检测Agent切换：AgentName变化时发射agent_start/agent_end
		if event.AgentName != "" && event.AgentName != currentAgent {
			if currentAgent != "" {
				if err := callback("agent_end", currentAgent); err != nil {
					return err
				}
			}
			currentAgent = event.AgentName
			if err := callback("agent_start", currentAgent); err != nil {
				return err
			}
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msgOutput := event.Output.MessageOutput

		// 检测工具调用请求：Assistant消息中包含ToolCalls（仅非流式完整消息）
		if !msgOutput.IsStreaming && msgOutput.Role == schema.Assistant &&
			msgOutput.Message != nil && len(msgOutput.Message.ToolCalls) > 0 {
			for _, tc := range msgOutput.Message.ToolCalls {
				if err := callback("tool_start", tc.Function.Name); err != nil {
					return err
				}
			}
		}

		// 检测工具执行结果
		if !msgOutput.IsStreaming && msgOutput.Role == schema.Tool && msgOutput.ToolName != "" {
			logger.Info("工具执行结果", logger.S("toolName", msgOutput.ToolName))
			if err := callback("tool_end", msgOutput.ToolName); err != nil {
				return err
			}
			if errMsg := extractToolError(msgOutput.Message.Content); errMsg != "" {
				payload := msgOutput.ToolName + "\x00" + errMsg
				if err := callback("tool_error", payload); err != nil {
					return err
				}
			}
		}

		if msgOutput.IsStreaming {
			for {
				chunk, err := msgOutput.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					logger.Error("流式读取失败", logger.C(err))
					return err
				}

				if chunk.ReasoningContent != "" {
					if err := callback("thinking", chunk.ReasoningContent); err != nil {
						return err
					}
					_, err = lastMsg.WriteString(chunk.Content)
					if err != nil {
						return err
					}
					continue
				}

				_, err = lastMsg.WriteString(chunk.Content)
				if err != nil {
					return err
				}

				if err := callback("response", chunk.Content); err != nil {
					return err
				}
			}
		} else {
			message := msgOutput.Message
			if message == nil {
				continue
			}
			// 仅纯文本Assistant消息（无ToolCalls）才发射response
			if msgOutput.Role == schema.Assistant && len(message.ToolCalls) == 0 {
				lastMsg.WriteString(message.Content)
				if err := callback("response", message.Content); err != nil {
					return err
				}
			}
		}
	}

	// 发送最后一个agent的结束事件
	if currentAgent != "" {
		if err := callback("agent_end", currentAgent); err != nil {
			return err
		}
	}

	logger.Debug("AI事件迭代完成", logger.I("eventCount", eventCount), logger.I("msgLen", lastMsg.Len()))

	if lastMsg.String() == "" {
		return errorer.New(errorer.ErrCallBackNull)
	}
	e.Humen.Memory.AppendAssistant(lastMsg.String())
	if err := e.Humen.Memory.Save(); err != nil {
		return err
	}
	return nil
}

// extractToolError 从工具返回的JSON结果中提取错误消息
func extractToolError(content string) string {
	var result struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ""
	}
	return result.Error
}
