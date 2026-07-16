package executor

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
)

const maxRetries = 3

// runResult agent 单次完整运行的结果。
type runResult struct {
	lastMsg      string // 最终助理回复文本
	currentAgent string // 最后一个活跃的 Agent 名称
	eventCount   int    // 处理的事件总数
}

// runConversation 构建 prompt 后运行 agent（含最多 maxRetries 次重试），返回最终结果。
func (e *Executor) runConversation(
	ctx context.Context,
	req *domain.TalkReq,
	toolCB callbacks.Handler,
	callback func(event, content string) error,
) (*runResult, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			logger.Warn(ctx, "AI调用失败，重试中",
				logger.I("attempt", attempt+1),
				logger.I("maxRetries", maxRetries))
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}

		msgs, err := e.Humen.Prompt.Build(ctx, req.Content)
		if err != nil {
			logger.Error(ctx, "构建提示词失败", logger.C(err))
			return nil, err
		}

		result, retry, err := e.runAgentOnce(ctx, msgs, toolCB, callback)
		if retry {
			continue
		}
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, errorer.New("AI调用达到最大重试次数")
}

// runAgentOnce 单次 agent 运行，迭代事件并收集最终助理回复。
// 返回值：result（成功时）、retry（可重试的错误）、err（致命错误）。
func (e *Executor) runAgentOnce(
	ctx context.Context,
	msgs []*schema.Message,
	toolCB callbacks.Handler,
	callback func(event, content string) error,
) (*runResult, bool, error) {
	e.Token.reset()

	iter := e.Runner.Run(ctx, msgs, adk.WithCallbacks(toolCB))
	lastMsg := &strings.Builder{}
	var currentAgent string
	eventCount := 0

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++

		if event.Err != nil {
			if errors.Is(event.Err, context.Canceled) {
				logger.Debug(ctx, "AI事件被取消（客户端断开）", logger.C(event.Err))
				return nil, false, context.Canceled
			}
			if isRetryable(event.Err) {
				logger.Warn(ctx, "AI调用临时错误，将重试", logger.C(event.Err))
				return nil, true, nil
			}
			logger.Error(ctx, "AI事件错误", logger.C(event.Err))
			return nil, false, event.Err
		}

		// 检测 Agent 切换
		if event.AgentName != "" && event.AgentName != currentAgent {
			if currentAgent != "" {
				_ = callback("agent_end", currentAgent)
			}
			currentAgent = event.AgentName
			_ = callback("agent_start", currentAgent)
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msgOutput := event.Output.MessageOutput
		if err := e.handleEvent(msgOutput, lastMsg, callback); err != nil {
			return nil, false, err
		}
	}

	// 发送最后一个 agent 的结束事件
	logger.Debug(ctx, "callback", logger.S("agent_end", currentAgent))
	if currentAgent != "" {
		_ = callback("agent_end", currentAgent)
	}

	return &runResult{
		lastMsg:      lastMsg.String(),
		currentAgent: currentAgent,
		eventCount:   eventCount,
	}, false, nil
}

// handleEvent 处理单个消息输出事件，分流为流式/非流式并写入 lastMsg。
func (e *Executor) handleEvent(
	msgOutput *adk.MessageVariant,
	lastMsg *strings.Builder,
	callback func(event, content string) error,
) error {
	if msgOutput.IsStreaming {
		return e.handleStreaming(msgOutput, lastMsg, callback)
	}
	return e.handleNonStreaming(msgOutput, lastMsg, callback)
}

// handleStreaming 处理流式消息输出：逐 chunk 读取并发射 response/thinking。
func (e *Executor) handleStreaming(
	msgOutput *adk.MessageVariant,
	lastMsg *strings.Builder,
	callback func(event, content string) error,
) error {
	for {
		chunk, err := msgOutput.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		if chunk.ReasoningContent != "" {
			_ = callback("thinking", chunk.ReasoningContent)
			lastMsg.WriteString(chunk.Content)
			if e.Token.accumulate(chunk) {
				_ = e.Token.send(callback)
				e.checkCompressionThreshold()
			}
			continue
		}

		lastMsg.WriteString(chunk.Content)
		_ = callback("response", chunk.Content)

		if e.Token.accumulate(chunk) {
			_ = e.Token.send(callback)
			e.checkCompressionThreshold()
		}
	}
}

// handleNonStreaming 处理非流式消息输出：只取纯文本 Assistant 消息。
func (e *Executor) handleNonStreaming(
	msgOutput *adk.MessageVariant,
	lastMsg *strings.Builder,
	callback func(event, content string) error,
) error {
	message := msgOutput.Message
	if message == nil {
		return nil
	}
	// 只收集最终 AI 回复（纯文本，无工具调用）
	if msgOutput.Role == schema.Assistant && len(message.ToolCalls) == 0 {
		lastMsg.WriteString(message.Content)
		_ = callback("response", message.Content)
	}
	// 累加 token
	if e.Token.accumulate(message) {
		_ = e.Token.send(callback)
		e.checkCompressionThreshold()
	}
	return nil
}

// isRetryable 判断错误是否可重试（网络超时、TLS 握手、连接拒绝等临时错误）
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
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
