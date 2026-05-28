package tui

import (
	"mifer/pkg/conf"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) streamCmd() tea.Cmd {
	if m.streamCh != nil {
		return listenStreamCmd(m.streamCh)
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.height < conf.GetConfig().Cli.Tui.MinHeight && m.height > 0 {
			m.height = conf.GetConfig().Cli.Tui.MinHeight
		}
		if m.width > conf.GetConfig().Cli.Tui.ContentMargin*2 {
			m.textarea.SetWidth(m.width - conf.GetConfig().Cli.Tui.ContentMargin*2)
		}
		m.adjustInputHeight()
		sidebarW := m.width / 4
		if sidebarW < 20 {
			sidebarW = 20
		}
		if sidebarW > 40 {
			sidebarW = 40
		}
		m.viewport.Width = m.width - sidebarW - 1 - 2
		if m.viewport.Width < 10 {
			m.viewport.Width = 10
		}
		m.viewport.Height = m.contentHeight
		m.sidebarVP.Width = sidebarW - 2
		if m.sidebarVP.Width < 10 {
			m.sidebarVP.Width = 10
		}
		m.memoryList.SetSize(sidebarW-4, 8)
		m.rebackList.SetSize(sidebarW-4, 8)
		m.confirmList.SetSize(sidebarW-4, 8)
		m.planList.SetSize(sidebarW-4, 8)
		m.memoryViewport.Width = m.width - 4
		m.memoryViewport.Height = m.height - 2
		_, _ = m.textarea.Update(msg)
		_, _ = m.viewport.Update(msg)
		_, _ = m.sidebarVP.Update(msg)
		_, _ = m.memoryList.Update(msg)
		_, _ = m.rebackList.Update(msg)
		_, _ = m.confirmList.Update(msg)
		_, _ = m.planList.Update(msg)
		return m, nil

	case tea.MouseMsg:
		if m.showingMemoryView {
			var cmd tea.Cmd
			m.memoryViewport, cmd = m.memoryViewport.Update(msg)
			return m, cmd
		}
		if msg.Button == tea.MouseButtonWheelLeft || msg.Button == tea.MouseButtonWheelRight ||
			(msg.Alt && msg.Button == tea.MouseButtonWheelUp) ||
			(msg.Alt && msg.Button == tea.MouseButtonWheelDown) {
			switch msg.Button {
			case tea.MouseButtonWheelLeft, tea.MouseButtonWheelUp:
				m.viewport.ScrollLeft(conf.GetConfig().Cli.Tui.HorizontalScrollStep)
			case tea.MouseButtonWheelRight, tea.MouseButtonWheelDown:
				m.viewport.ScrollRight(conf.GetConfig().Cli.Tui.HorizontalScrollStep)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.showingMemoryView {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.showingMemoryView = false
				m.memoryViewContent = ""
				return m, nil
			default:
				var cmd tea.Cmd
				m.memoryViewport, cmd = m.memoryViewport.Update(msg)
				return m, cmd
			}
		}

		if m.selectingMem {
			switch msg.String() {
			case "enter":
				return m.handleMemorySelect()
			case "esc":
				m.selectingMem = false
				m.pendingMemCmd = ""
				return m, nil
			case "up", "down", "k", "j", "home", "end", "pgup", "pgdown":
				var cmd tea.Cmd
				m.memoryList, cmd = m.memoryList.Update(msg)
				return m, cmd
			default:
				return m, nil
			}
		}

		if m.planSupplement {
			switch msg.String() {
			case "enter":
				return m.handlePlanSupplementSubmit()
			case "esc":
				m.planSupplement = false
				m.confirmingPlan = false
				m.textarea.Reset()
				m.textarea.Placeholder = ""
				m.textarea.InsertString(m.pendingPlanInput)
				m.adjustInputHeight()
				return m, tea.Batch(planConfirmCmd(m.client, m.planConfirmCallID, "refuse", ""), m.streamCmd())
			default:
				m.textarea, _ = m.textarea.Update(msg)
				m.adjustInputHeight()
				return m, m.streamCmd()
			}
		}

		if m.confirmingPlan {
			switch msg.String() {
			case "enter":
				return m.handlePlanConfirmSelect()
			case "esc":
				m.confirmingPlan = false
				return m, tea.Batch(planConfirmCmd(m.client, m.planConfirmCallID, "refuse", ""), m.streamCmd())
			case "up", "down", "k", "j":
				var cmd tea.Cmd
				m.planList, cmd = m.planList.Update(msg)
				return m, tea.Batch(cmd, m.streamCmd())
			default:
				return m, m.streamCmd()
			}
		}

		if m.confirmingTool {
			switch msg.String() {
			case "enter":
				return m.handleConfirmSelect()
			case "esc":
				return m.resolveCurrentConfirm("refuse")
			case "up", "down", "k", "j":
				var cmd tea.Cmd
				m.confirmList, cmd = m.confirmList.Update(msg)
				return m, tea.Batch(cmd, m.streamCmd())
			default:
				return m, m.streamCmd()
			}
		}

		if m.selectingReback {
			switch msg.String() {
			case "enter":
				return m.handleRebackSelect()
			case "esc":
				m.selectingReback = false
				return m, nil
			case "up", "down", "k", "j", "home", "end", "pgup", "pgdown":
				var cmd tea.Cmd
				m.rebackList, cmd = m.rebackList.Update(msg)
				return m, cmd
			default:
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			if m.showingCompletions {
				m.showingCompletions = false
				m.resetCompletion()
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+n":
			m.textarea.InsertString("\n")
			m.adjustInputHeight()
			return m, nil
		case "enter":
			if m.showingCompletions && m.completionIdx >= 0 && m.completionIdx < len(m.completions) {
				m.textarea.Reset()
				m.textarea.InsertString(m.completions[m.completionIdx] + " ")
				m.textarea.CursorEnd()
				m.resetCompletion()
				m.showingCompletions = false
				m.adjustInputHeight()
				return m, nil
			}
			return m.handleEnter()
		case "up":
			if m.showingCompletions && m.completionIdx > 0 {
				m.completionIdx--
				return m, nil
			}
			if m.textarea.Line() == 0 {
				return m.handleHistoryUp()
			}
			m.textarea, _ = m.textarea.Update(msg)
			m.adjustInputHeight()
			return m, nil
		case "down":
			if m.showingCompletions && m.completionIdx < len(m.completions)-1 {
				m.completionIdx++
				return m, nil
			}
			if m.textarea.Line() == m.textarea.LineCount()-1 {
				return m.handleHistoryDown()
			}
			m.textarea, _ = m.textarea.Update(msg)
			m.adjustInputHeight()
			return m, nil
		case "tab":
			return m.handleTabComplete()
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			m.adjustInputHeight()
			if m.completionIdx != -1 && m.textarea.Value() != m.completionBase && !strings.HasPrefix(m.textarea.Value(), m.completionBase) {
				m.resetCompletion()
			}
			m.updateCompletionForInput()
			return m, cmd
		}

	case streamStatusMsg:
		return m.handleStreamStatus(msg)

	case streamContentMsg:
		return m.handleStreamContent(msg)

	case streamDoneMsg:
		return m.handleStreamDone(msg)

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
		m.needsAutoScroll = true
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
		m.needsAutoScroll = true
		return m, nil

	case memoryListMsg:
		return m.handleMemoryList(msg)

	case memoryViewMsg:
		return m.handleMemoryView(msg)

	case rebackListMsg:
		return m.handleRebackList(msg)

	case toolConfirmMsg:
		return m.handleToolConfirm(msg)

	case planConfirmMsg:
		return m.handlePlanConfirm(msg)

	case rebackDoneMsg:
		return m.handleRebackDone(msg)

	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) adjustInputHeight() {
	lines := max(m.textarea.LineCount(), 1)
	m.textarea.SetHeight(lines)
	m.contentHeight = max(m.height-m.textarea.Height()-1, 1)
}

func (m *Model) updateCompletionForInput() {
	input := strings.TrimSpace(m.textarea.Value())
	if !strings.HasPrefix(input, "/") {
		if m.showingCompletions {
			m.showingCompletions = false
			m.resetCompletion()
		}
		return
	}
	matches := m.findMatches(input)
	if len(matches) == 0 {
		if m.showingCompletions {
			m.showingCompletions = false
			m.resetCompletion()
		}
		return
	}
	m.completions = matches
	m.showingCompletions = true
	if m.completionIdx == -1 || m.completionIdx >= len(matches) {
		m.completionIdx = -1
	}
}

func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	m.textarea.Reset()
	m.textarea.SetHeight(1)
	m.contentHeight = max(m.height-1-1, 1)
	m.viewport.Height = m.contentHeight
	m.err = ""

	if len(m.history) == 0 || m.history[len(m.history)-1] != input {
		if len(m.history) >= conf.GetConfig().Cli.Tui.MaxHistory {
			m.history = m.history[1:]
		}
		m.history = append(m.history, input)
	}
	m.historyIdx = -1
	m.pendingInput = ""
	m.resetCompletion()

	switch {
	case input == "/exit" || input == "/quit":
		return m, tea.Quit

	case input == "/help":
		m.messages = append(m.messages, message{
			role:    "system",
			content: "命令: ↑↓ 历史输入 | Ctrl+N 换行 | /viewmemory 查看记忆 | /excmem <id> 切换会话 | /reback 回退对话 | /clear 新建会话 | /plan <任务> 制定计划 | /prompt 查看/设置提示词 | /reload 重载配置 | /exit 退出 | /help 帮助",
		})
		m.needsAutoScroll = true
		return m, nil

	case input == "/clear":
		return m, clearCmd(m.client)

	case input == "/reload":
		return m, reloadCmd(m.client)

	case input == "/prompt":
		return m, promptGetCmd(m.client)

	case strings.HasPrefix(input, "/prompt "):
		text := strings.TrimSpace(strings.TrimPrefix(input, "/prompt"))
		if text == "reset" {
			return m, promptResetCmd(m.client)
		}
		return m, promptSetCmd(m.client, text)

	case strings.HasPrefix(input, "/viewmemory"):
		id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory"))
		return m, listMemoriesCmd(m.client, "/viewmemory", id)

	case strings.HasPrefix(input, "/excmem"):
		id := strings.TrimSpace(strings.TrimPrefix(input, "/excmem"))
		return m, listMemoriesCmd(m.client, "/excmem", id)

	case input == "/reback":
		return m, listRebackEntriesCmd(m.client)

	case strings.HasPrefix(input, "/plan "):
		task := strings.TrimSpace(strings.TrimPrefix(input, "/plan "))
		m.messages = append(m.messages, message{
			role:    "user",
			content: "/plan " + task,
		})
		m.thinking = true
		m.needsAutoScroll = true
		m.accBuf = &strings.Builder{}
		m.streamCh = make(chan tea.Msg, 32)
		m.sidebar = SidebarState{}
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(m.spinner.Tick())
		return m, tea.Batch(
			startPlanSSECmd(m.client, task, m.streamCh),
			listenStreamCmd(m.streamCh),
			spCmd,
		)

	default:
		m.messages = append(m.messages, message{
			role:    "user",
			content: input,
		})
		m.thinking = true
		m.needsAutoScroll = true
		m.accBuf = &strings.Builder{}
		m.streamCh = make(chan tea.Msg, 32)
		m.sidebar = SidebarState{}
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(m.spinner.Tick())
		return m, tea.Batch(
			startSSECmd(m.client, input, m.streamCh),
			listenStreamCmd(m.streamCh),
			spCmd,
		)
	}
}

func (m *Model) handleHistoryUp() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	if m.historyIdx == -1 {
		m.pendingInput = m.textarea.Value()
		m.historyIdx = len(m.history) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}
	m.textarea.Reset()
	m.textarea.InsertString(m.history[m.historyIdx])
	m.textarea.CursorEnd()
	m.adjustInputHeight()
	return m, nil
}

func (m *Model) handleHistoryDown() (tea.Model, tea.Cmd) {
	if m.historyIdx == -1 {
		return m, nil
	}
	if m.historyIdx < len(m.history)-1 {
		m.historyIdx++
		m.textarea.Reset()
		m.textarea.InsertString(m.history[m.historyIdx])
		m.textarea.CursorEnd()
	} else {
		m.historyIdx = -1
		m.textarea.Reset()
		m.textarea.InsertString(m.pendingInput)
		m.textarea.CursorEnd()
		m.pendingInput = ""
	}
	m.adjustInputHeight()
	return m, nil
}

func (m *Model) handleTabComplete() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()
	trimmed := strings.TrimSpace(input)

	if m.completionIdx >= 0 && trimmed == m.completionBase {
		return m.cycleCompletion()
	}

	matches := m.findMatches(trimmed)
	switch len(matches) {
	case 0:
		return m, nil
	case 1:
		m.textarea.Reset()
		m.textarea.InsertString(matches[0] + " ")
		m.textarea.CursorEnd()
		m.resetCompletion()
		m.adjustInputHeight()
		return m, nil
	default:
		common := longestCommonPrefix(matches)
		if common != trimmed {
			m.textarea.Reset()
			m.textarea.InsertString(common)
			m.textarea.CursorEnd()
		}
		m.completions = matches
		m.completionIdx = -1
		m.completionBase = common
		m.adjustInputHeight()
		return m, nil
	}
}

func (m *Model) cycleCompletion() (tea.Model, tea.Cmd) {
	if len(m.completions) == 0 {
		return m, nil
	}
	m.completionIdx = (m.completionIdx + 1) % len(m.completions)
	m.textarea.Reset()
	m.textarea.InsertString(m.completions[m.completionIdx] + " ")
	m.textarea.CursorEnd()
	m.adjustInputHeight()
	return m, nil
}

func (m *Model) findMatches(prefix string) []string {
	if prefix == "" {
		return nil
	}
	lower := strings.ToLower(prefix)
	var matches []string
	for _, cmd := range conf.GetConfig().Cli.Tui.CompletableCommands {
		if strings.HasPrefix(strings.ToLower(cmd), lower) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

func (m *Model) resetCompletion() {
	m.completions = nil
	m.completionIdx = -1
	m.completionBase = ""
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}
