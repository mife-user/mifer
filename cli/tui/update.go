package tui

import (
	"context"
	"strings"
	"time"

	"mifer/cli/client"

	tea "github.com/charmbracelet/bubbletea"
)

// Update 消息处理
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.height < minHeight && m.height > 0 {
			m.height = minHeight
		}
		if m.width > contentMargin*2 {
			m.textarea.SetWidth(m.width - contentMargin*2)
		}
		// 动态 textarea 高度
		taHeight := m.height / 6
		if taHeight < 1 {
			taHeight = 1
		}
		m.textarea.SetHeight(taHeight)
		// 传递 resize 给 textarea
		_, _ = m.textarea.Update(msg)
		return m, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.scrollOff > 0 {
				m.scrollOff--
			}
		case tea.MouseButtonWheelDown:
			m.scrollOff++
			// 超出部分由 View 限制
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}
			m.textarea.Reset()
			m.err = ""

			switch {
			case input == "exit" || input == "quit":
				return m, tea.Quit
			case input == "help":
				m.messages = append(m.messages, message{
					role:    "system",
					content: "命令: /viewmemory 查看记忆 | /excmem <id> 切换会话 | exit 退出 | help 帮助",
				})
				return m, nil
			case strings.HasPrefix(input, "/viewmemory"):
				id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory"))
				return m, loadMemoryCmd(m.client, id)
			case strings.HasPrefix(input, "/excmem"):
				id := strings.TrimSpace(strings.TrimPrefix(input, "/excmem"))
				return m, excmemCmd(m.client, id)
			default:
				// 用户聊天消息
				m.messages = append(m.messages, message{
					role:    "user",
					content: input,
				})
				m.thinking = true
				return m, tea.Batch(
					sendChatCmd(m.client, input),
					thinkingTickCmd(),
				)
			}

		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case chatRespMsg:
		m.thinking = false
		if msg.err != nil {
			m.err = "错误: " + msg.err.Error()
			return m, nil
		}
		content := strings.TrimSpace(msg.content)
		if content == "" {
			m.err = "AI 返回了空内容"
			return m, nil
		}
		// 通过 glamour 渲染 markdown
		rendered, err := m.mark.Render(content)
		if err != nil {
			m.messages = append(m.messages, message{
				role:    "assistant",
				content: content,
			})
			m.err = "Markdown 渲染失败，显示原始内容"
			return m, nil
		}
		m.messages = append(m.messages, message{
			role:     "assistant",
			content:  content,
			rendered: rendered,
		})
		return m, nil

	case systemMsg:
		if msg.err != nil {
			m.err = "错误: " + msg.err.Error()
			return m, nil
		}
		m.messages = append(m.messages, message{
			role:    "system",
			content: msg.content,
		})
		return m, nil

	case thinkingTickMsg:
		if m.thinking {
			m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
			return m, thinkingTickCmd()
		}
		return m, nil
	}

	return m, nil
}

// sendChatCmd 发送聊天请求，积累所有 SSE chunk 后返回完整响应
func sendChatCmd(client *client.Client, content string) tea.Cmd {
	return func() tea.Msg {
		var buf strings.Builder
		ctx := context.Background()
		err := client.Chat.Send(ctx, content, func(chunk string) error {
			buf.WriteString(chunk)
			return nil
		})
		return chatRespMsg{content: buf.String(), err: err}
	}
}

// thinkingTickCmd 思考动画 tick
// 在 tea.Batch 内部执行 Tea 的 Tick 命令并返回其结果
func thinkingTickCmd() tea.Cmd {
	tickCmd := tea.Tick(thinkingTickInterval, func(_ time.Time) tea.Msg {
		return thinkingTickMsg{}
	})
	return func() tea.Msg {
		return tickCmd()
	}
}

// loadMemoryCmd 加载记忆命令
func loadMemoryCmd(client *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if id == "" {
			id = "default"
		}
		memory, err := client.Memory.Load(id)
		if err != nil {
			return systemMsg{err: err}
		}
		if memory == "" {
			memory = "(暂无对话记忆)"
		}
		return systemMsg{content: "═══════ 对话记忆 ═══════\n" + memory + "\n══════════════════════════"}
	}
}

// excmemCmd 切换记忆会话命令
func excmemCmd(client *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Excmem.Exchange(id); err != nil {
			return systemMsg{err: err}
		}
		return systemMsg{content: "已切换到记忆会话: " + id}
	}
}
