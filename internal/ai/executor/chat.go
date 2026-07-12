package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mifer/internal/ai/agent"
	aicallback "mifer/internal/ai/callback"
	"mifer/internal/ai/confirm"
	"mifer/internal/domain"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const maxRetries = 3

// Chat 执行一次对话，通过 callback 将事件实时传递到上层。
// tool_start / tool_end / tool_error 由 aicallback.NewHandler 通过 adk.WithCallbacks 按调用注入，
// 此处仅处理对话内容（response、thinking）、Agent 切换和 token 统计。
// 网络临时错误（TLS 超时等）自动重试最多 maxRetries 次。
func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(event, content string) error) error {
	// 防止并发 Chat 调用导致 Memory 数据竞争
	e.chatMu.Lock()
	defer e.chatMu.Unlock()

	// 检查模型是否可用（api_key 未配置时 Runner 为 nil）
	if e.Runner == nil {
		callback("response", "当前apikey未配置，AI对话功能暂不可用。请使用 /config 命令编辑配置文件，在 ai.backends.default.api_key 填入您的API Key后重载配置。")
		return nil
	}

	// 检查是否需要压缩上下文（上一轮结束时已标记）
	if e.compress.needsCompression {
		if err := e.compress.compressor.Compress(
			c, e.Humen.Prompt.Memory, e.Humen.Prompt.SystemPrompt,
			e.compress.lastPromptTokens, callback,
		); err != nil {
			logger.Error("上下文压缩失败", logger.C(err))
		}
		e.compress.needsCompression = false
	}

	// 指定了 sessionID 时先切换记忆，保证 switch + chat 原子执行
	if req.SessionID != "" {
		if err := e.Humen.Prompt.Memory.SwitchSession(req.SessionID); err != nil {
			logger.Error("切换记忆会话失败", logger.S("session", req.SessionID), logger.C(err))
		}
	}

	e.Humen.Prompt.Memory.AppendUser(req.Content)

	// 获取会话 ID 用于工具确认和清理
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID, _ = c.Value("id").(string)
	}

	// 构建 per-invocation Tool 回调处理器，将 callback 通过闭包注入
	toolCBHandler := aicallback.NewHandler(callback)

	// 将 executor 回调注入 confirm 中间件的 context key
	ctx := confirm.WithCallback(c, confirm.ExecutorCallback(callback))

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
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		msgs, err := e.Humen.Prompt.Build(c, req.Content)
		if err != nil {
			logger.Error("构建提示词失败", logger.C(err))
			return err
		}
		// QQ 通道使用专用 Agent（无工具，纯文本），避免工具确认死循环
		runner := e.Runner
		if req.Channel == "qq" && e.QQRunner != nil {
			runner = e.QQRunner
		}
		iter := runner.Run(ctx, msgs, adk.WithCallbacks(toolCBHandler))

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
							e.checkCompressionThreshold()
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
						e.checkCompressionThreshold()
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
					e.checkCompressionThreshold()
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
			logger.Error("保存记忆失败", logger.C(err))
			return err
		}
		// 轮次结束后保存文件快照（需在配置中启用 snapshot_enabled）
		if e.Snapshot != nil {
			round := e.Humen.Prompt.Memory.CountUserMessages()
			logger.Debug("开始保存文件快照", logger.I("round", round))
			if err := e.Snapshot.SaveRound(round); err != nil {
				logger.Warn("保存文件快照失败（不中断对话流程）", logger.C(err), logger.I("round", round))
			} else {
				logger.Debug("文件快照保存完成", logger.I("round", round))
			}
		}

		// 异步总结用户习惯（fire-and-forget，不阻塞对话响应）
		if e.Humen.HabitGraph != nil {
			go e.summarizeHabits(req.Content, lastMsg.String())
		}
		return nil
	}

	return errorer.New("AI调用达到最大重试次数")
}

// summarizeHabits 异步调用习惯总结图，分析本轮对话并更新用户级 MIFER.md。
// 使用 context.Background() 独立于请求上下文，fire-and-forget 模式。
func (e *Executor) summarizeHabits(userMsg, assistantReply string) {
	ctx := context.Background()

	// 读取现有 MIFER.md（文件不存在时为空，AI 会生成初始画像）
	miferPath := filepath.Join(conf.GetConfig().Path.CfgPath, "MIFER.md")
	existingContent := ""
	if data, err := os.ReadFile(miferPath); err == nil {
		existingContent = string(data)
	}

	// 构建图输入消息
	msgs := []*schema.Message{
		schema.SystemMessage(agent.HabitInstruction),
		schema.UserMessage(fmt.Sprintf(
			"## 本轮对话\n用户: %s\n\nAI: %s\n\n## 现有用户画像\n%s",
			userMsg, assistantReply, existingContent,
		)),
	}

	if _, err := e.Humen.HabitGraph.Invoke(ctx, msgs); err != nil {
		logger.Warn("用户习惯总结失败", logger.C(err))
	}
}

// checkCompressionThreshold 检查当前 PromptTokens 是否超过压缩阈值
// 超过时设置 needsCompression 标记，供下一轮对话开始时压缩
func (e *Executor) checkCompressionThreshold() {
	if e.compress.needsCompression {
		return
	}
	ctxCfg := conf.GetConfig().Ai.Context
	if ctxCfg.Length <= 0 {
		return
	}
	thresholdTokens := int(float64(ctxCfg.Length) * ctxCfg.Threshold)
	if e.Token.Prompt >= thresholdTokens {
		e.compress.needsCompression = true
		e.compress.lastPromptTokens = e.Token.Prompt
		logger.Warn("上下文超过压缩阈值",
			logger.I("prompt_tokens", e.Token.Prompt),
			logger.I("threshold", thresholdTokens),
			logger.I("limit", ctxCfg.Length),
		)
	}
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
