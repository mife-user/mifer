package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	e.Humen.Prompt.Memory.AppendUser(req.Content)

	iter := e.Runner.Run(c, e.Humen.Prompt.Build())

	lastMsg := &strings.Builder{}
	eventCount := 0
	var currentAgent string // 跟踪当前执行的Agent，用于检测切换
	// Token 累计统计
	e.Token.reset()
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
					if e.Token.accumulate(chunk) {
						if err := e.Token.send(callback); err != nil {
							return err
						}
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

				if e.Token.accumulate(chunk) {
					if err := e.Token.send(callback); err != nil {
						return err
					}
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
			// 累加 token 用量（非流式消息）
			if e.Token.accumulate(message) {
				if err := e.Token.send(callback); err != nil {
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
	e.Humen.Prompt.Memory.AppendAssistant(lastMsg.String())
	if err := e.Humen.Prompt.Memory.Save(); err != nil {
		return err
	}
	return nil
}

// TokenUsage token 累计用量统计
type TokenUsage struct {
	Prompt     int // 输入 token
	Completion int // 输出 token
	Total      int // 合计 token
	Cached     int // 缓存命中 token
	Reasoning  int // 推理 token
}

// accumulate 从 schema.Message 中累加 token 用量，返回是否有新数据
func (t *TokenUsage) accumulate(msg *schema.Message) bool {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return false
	}
	usage := msg.ResponseMeta.Usage
	t.Prompt += usage.PromptTokens
	t.Completion += usage.CompletionTokens
	t.Total += usage.TotalTokens
	t.Cached += usage.PromptTokenDetails.CachedTokens
	t.Reasoning += usage.CompletionTokensDetails.ReasoningTokens
	return true
}

// send 发送 token 事件到回调
func (t *TokenUsage) send(callback func(event, content string) error) error {
	payload := fmt.Sprintf("%d\x00%d\x00%d\x00%d\x00%d",
		t.Prompt, t.Completion, t.Total, t.Cached, t.Reasoning)
	return callback("token", payload)
}

// reset 重置所有计数为零
func (t *TokenUsage) reset() {
	t.Prompt = 0
	t.Completion = 0
	t.Total = 0
	t.Cached = 0
	t.Reasoning = 0
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
