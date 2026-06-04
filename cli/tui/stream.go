package tui

// ============================================================================
// stream.go — 流式传输类型与逻辑
// ============================================================================
//
// 流式传输是 TUI 与后端 AI 服务的主要通信方式：
//   1. 用户提交消息 → handleEnter() 启动 startSSECmd（HTTP SSE 请求）
//   2. 后台 goroutine 将 SSE 事件写入 streamCh 通道
//   3. listenStreamCmd 阻塞读取通道，将事件送入 Update()
//   4. Update() 分发到对应 handler：
//      - streamStatusMsg → handleStreamStatus（更新侧边栏）
//      - streamContentMsg → handleStreamContent（累积到 accBuf）
//      - streamDoneMsg → handleStreamDone（markdown 渲染并追加消息）

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"mifer/cli/client"
	"mifer/pkg/exc"
	"mifer/pkg/logger"

	tea "github.com/charmbracelet/bubbletea"
)

// TokenUsageData token 消耗统计
type TokenUsageData struct {
	PromptTokens     int // 输入 token
	CompletionTokens int // 输出 token
	TotalTokens      int // 合计 token
	CachedTokens     int // 缓存命中 token
	ReasoningTokens  int // 推理 token
}

// streamStatusMsg AI流式响应中的状态更新（agent切换、工具调用、工具错误、token统计、系统通知）
type streamStatusMsg struct {
	event      string          // "agent_start" | "agent_end" | "tool_start" | "tool_end" | "tool_error" | "token" | "system"
	name       string          // agent名称或工具名称
	arg        string          // tool_error 时携带的错误消息 或 tool_start 时的工具参数 JSON
	tokenUsage *TokenUsageData // token 事件时的数据（nil 表示非 token 事件）
}

// streamContentMsg AI流式响应中的内容片段
type streamContentMsg struct {
	content string
}

// streamDoneMsg AI流式传输完成
type streamDoneMsg struct {
	err error
}

// ============================================================================
// Update() 中的流式消息处理器
// ============================================================================

// handleStreamStatus 处理流式状态更新
// 同时更新侧边栏和对话框：agent/tool 开始结束时在对话框中显示相应信息，
// agent 结束时立即提取并渲染该 agent 的增量内容
func (m *Model) handleStreamStatus(msg streamStatusMsg) (tea.Model, tea.Cmd) {
	m.sidebar.update(msg)

	switch msg.event {
	case "agent_start":
		m.messages = append(m.messages, message{
			role:    "system",
			content: msg.name + " begin",
		})
		m.agentContentStart = m.accBuf.Len()
		m.needsAutoScroll = true

	case "agent_end":
		content := strings.TrimSpace(m.accBuf.String()[m.agentContentStart:])
		m.agentContentStart = m.accBuf.Len()
		if content != "" {
			rendered, err := m.mark.Render(content)
			if err != nil {
				m.messages = append(m.messages, message{
					role:    "assistant",
					content: content,
				})
			} else {
				m.messages = append(m.messages, message{
					role:     "assistant",
					content:  content,
					rendered: rendered,
				})
			}
			m.needsAutoScroll = true
		}

		case "tool_start":
			// 仅更新侧边栏，不在对话框显示（避免与 tool_confirm 重复）
			logger.Info("调用工具", logger.S("name", msg.name), logger.S("arg", msg.arg))

		case "tool_end":
			// 仅更新侧边栏

		case "tool_error":
			m.messages = append(m.messages, message{
				role:    "system",
				content: "--- " + msg.name + " error: " + msg.arg,
			})
			m.needsAutoScroll = true

		case "system":
			m.messages = append(m.messages, message{
				role:    "system",
				content: msg.name,
			})
			m.needsAutoScroll = true
	}

	if m.streamCh != nil {
		return m, listenStreamCmd(m.streamCh)
	}
	return m, nil
}

// handleStreamContent 处理流式内容片段，累积到 accBuf
func (m *Model) handleStreamContent(msg streamContentMsg) (tea.Model, tea.Cmd) {
	if m.accBuf != nil {
		m.accBuf.WriteString(msg.content)
	}
	if m.streamCh != nil {
		return m, listenStreamCmd(m.streamCh)
	}
	return m, nil
}

// handleStreamDone 流式传输完成
// 大部分内容已在 agent_end 时增量渲染，此处仅处理最后残留的尾部内容
func (m *Model) handleStreamDone(msg streamDoneMsg) (tea.Model, tea.Cmd) {
	m.thinking = false
	m.streamCh = nil

	if msg.err != nil {
		// 用户主动取消（Ctrl+C / Esc），静默清理不显示错误
		if errors.Is(msg.err, context.Canceled) {
			m.accBuf = nil
			m.sidebar = SidebarState{}
			return m, nil
		}
		m.err = "错误: " + msg.err.Error()
		m.accBuf = nil
		m.sidebar = SidebarState{}
		return m, nil
	}

	// 处理 agent_end 之后可能残留的尾部内容
	if m.accBuf != nil && m.accBuf.Len() > m.agentContentStart {
		content := strings.TrimSpace(m.accBuf.String()[m.agentContentStart:])
		if content != "" {
			rendered, err := m.mark.Render(content)
			if err != nil {
				m.messages = append(m.messages, message{
					role:    "assistant",
					content: content,
				})
			} else {
				m.messages = append(m.messages, message{
					role:     "assistant",
					content:  content,
					rendered: rendered,
				})
			}
		}
	}

	m.accBuf = nil
	m.needsAutoScroll = true
	return m, nil
}

// ============================================================================
// 流式传输命令
// ============================================================================

// formatToolArgs 格式化工具参数 JSON，美化并截断
func formatToolArgs(rawJSON string) string {
	if rawJSON == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		if len(rawJSON) > 120 {
			return rawJSON[:120] + "..."
		}
		return rawJSON
	}
	compact, err := json.Marshal(parsed)
	if err != nil {
		return rawJSON
	}
	s := string(compact)
	if len(s) > 150 {
		return s[:150] + "..."
	}
	return s
}

// startSSECmd 启动 SSE 流式请求，将事件写入通道
//
// 返回的 tea.Cmd 在后台 goroutine 中执行 SSE 请求。
// 通过传入的 channel 逐条推送流式消息回 Bubble Tea 事件循环。
//
// 流式消息分发逻辑：
//
//	agent_start/agent_end/tool_start/tool_end → streamStatusMsg（侧边栏更新）
//	tool_error → streamStatusMsg（工具错误）
//	system → streamStatusMsg（系统通知）
//	token → streamStatusMsg（token 统计）
//	response → streamContentMsg（累积到 accBuf）
//	thinking → 跳过
//
// goroutine 退出前关闭 channel，触发 listenStreamCmd 返回 nil 停止递归。
func startSSECmd(client *client.Client, content string, ch chan<- tea.Msg, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer close(ch)
			err := client.Chat.Send(ctx, content, func(event, chunk string) error {
				switch event {
				case "agent_start", "agent_end":
					ch <- streamStatusMsg{event: event, name: chunk}
				case "tool_start":
					name, args, _ := strings.Cut(chunk, "\x00")
					ch <- streamStatusMsg{event: "tool_start", name: name, arg: args}
				case "tool_end":
					ch <- streamStatusMsg{event: "tool_end", name: chunk}
				case "tool_error":
					if name, errMsg, ok := strings.Cut(chunk, "\x00"); ok {
						ch <- streamStatusMsg{event: "tool_error", name: name, arg: errMsg}
					}
				case "token":
					parts := strings.Split(chunk, "\x00")
					if len(parts) == 5 {
						usage := &TokenUsageData{}
						usage.PromptTokens, _ = strconv.Atoi(parts[0])
						usage.CompletionTokens, _ = strconv.Atoi(parts[1])
						usage.TotalTokens, _ = strconv.Atoi(parts[2])
						usage.CachedTokens, _ = strconv.Atoi(parts[3])
						usage.ReasoningTokens, _ = strconv.Atoi(parts[4])
						ch <- streamStatusMsg{event: "token", tokenUsage: usage}
					}
				case "tool_confirm":
					var prompt ToolConfirmPrompt
					if err := exc.ExcJSONToFile(chunk, &prompt); err == nil {
						ch <- toolConfirmMsg{prompt: &prompt}
					} else {
						logger.Error("tool_confirm JSON解析失败",
							logger.C(err),
							logger.S("chunk_preview", chunk[:min(len(chunk), 200)]))
					}
				case "system":
					ch <- streamStatusMsg{event: "system", name: chunk}
				case "thinking":
					// 跳过 thinking 事件
				case "response":
					ch <- streamContentMsg{content: chunk}
				}
				return nil
			})
			ch <- streamDoneMsg{err: err}
		}()
		return nil // nil msg 被 Bubble Tea 忽略
	}
}

// listenStreamCmd 从通道读取下一条流式消息的递归命令
//
// Bubble Tea 的标准"持续监听"模式：
//
//	收到消息 → 在 Update 中处理后返回 listenStreamCmd(ch)
//	→ 阻塞等待下一条消息 → 收到后再次进入 Update → 循环
//	通道关闭时返回 nil（递归终止），后续不再有 stream 消息
func listenStreamCmd(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil // 通道关闭，递归终止
		}
		return msg
	}
}
