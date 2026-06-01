package executor

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	aicallback "mifer/internal/ai/callback"
	"mifer/internal/ai/confirm"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

const maxRetries = 3

// Chat 执行一次对话，通过 callback 将事件实时传递到上层。
// tool_start / tool_end / tool_error 已由 aicallback.ToolCallbackHandler 统一拦截，
// 此处仅处理对话内容（response、thinking）、Agent 切换和 token 统计。
// 网络临时错误（TLS 超时等）自动重试最多 maxRetries 次。
func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	e.Humen.Prompt.Memory.AppendUser(req.Content)

	// 获取会话 ID 用于工具确认和清理
	sessionID, _ := c.Value("id").(string)

	// 将 executor 回调注入 context，供 Eino callback handler 捕获 Tool 调用
	ctx := aicallback.WithExecutorCallback(c, callback)

	// 将 executor 回调注入 confirm 中间件的 context key
	ctx = confirm.WithCallback(ctx, confirm.ExecutorCallback(callback))

	// 将会话 ID 注入 context，供 confirm 中间件使用
	ctx = confirm.WithSessionID(ctx, sessionID)

	// 确保对话结束时清理该 session 的所有待确认项
	defer e.Humen.ConfirmStore.Cleanup(sessionID)

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			logger.Warn("AI调用失败，重试中",
				logger.I("attempt", attempt+1),
				logger.I("maxRetries", maxRetries))
			// 递增等待：1s / 2s / 3s
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		msgs, err := e.Humen.Prompt.Build(c, req.Content)
		if err != nil {
			return err
		}
		iter := e.Runner.Run(ctx, msgs)

		lastMsg := &strings.Builder{}
		eventCount := 0
		var currentAgent string
		var retry bool
		e.Token.reset()
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			eventCount++
			if event.Err != nil {
				if errors.Is(event.Err, context.Canceled) {
					logger.Debug("AI事件被取消（客户端断开）", logger.C(event.Err))
					return nil
				}
				if attempt < maxRetries-1 && isRetryable(event.Err) {
					logger.Warn("AI调用临时错误，将重试", logger.C(event.Err))
					retry = true
					break
				}
				logger.Error("AI事件错误", logger.C(event.Err))
				return event.Err
			}

			// 检测 Agent 切换
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
				// 仅纯文本 Assistant 消息（无 ToolCalls）才发射 response
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

		if retry {
			continue
		}

		// 发送最后一个 agent 的结束事件
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

	return errorer.New("AI调用达到最大重试次数")
}

// isRetryable 判断错误是否可重试（网络超时、TLS 握手、连接拒绝等临时错误）
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// 网络超时 / TLS 握手 / 连接问题
	keywords := []string{
		"timeout",
		"TLS handshake",
		"connection refused",
		"connection reset",
		"EOF",
		"no such host",
		"i/o timeout",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
