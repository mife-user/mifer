package executor

import (
	"context"
	"errors"
	"io"
	"strings"

	aicallback "mifer/internal/ai/callback"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

// Chat 执行一次对话，通过 callback 将事件实时传递到上层。
// tool_start / tool_end / tool_error 已由 aicallback.ToolCallbackHandler 统一拦截，
// 此处仅处理对话内容（response、thinking）、Agent 切换和 token 统计。
func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	e.Humen.Prompt.Memory.AppendUser(req.Content)

	// 将 executor 回调注入 context，供 Eino callback handler 捕获 Tool 调用
	ctx := aicallback.WithExecutorCallback(c, callback)

	msgs, err := e.Humen.Prompt.Build(c, req.Content)
	if err != nil {
		return err
	}
	iter := e.Runner.Run(ctx, msgs)

	lastMsg := &strings.Builder{}
	eventCount := 0
	var currentAgent string
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
