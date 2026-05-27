package tui

import (
	"fmt"
	"strings"

	"mifer/pkg/conf"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.showingMemoryView {
		title := m.lip.SidebarActive.Render(" 对话记忆 — Esc 返回")
		sep := m.lip.SidebarSeparator.Render(strings.Repeat("─", m.width-4))
		return lipgloss.JoinVertical(lipgloss.Top, title, sep, m.memoryViewport.View())
	}

	if m.width == 0 {
		return "正在启动..."
	}
	if m.contentHeight < 1 {
		return fmt.Sprintf("窗口太小 (当前 %d, 最少 %d)", m.height, conf.GetConfig().Cli.Tui.MinHeight)
	}

	sidebarWidth := m.width / 4
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}

	var msgLines []string
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			msgLines = append(msgLines, m.lip.User.Render("You: "+msg.content))

		case "assistant":
			if msg.rendered != "" {
				msgLines = append(msgLines, strings.Split(msg.rendered, "\n")...)
			} else {
				msgLines = append(msgLines, msg.content)
			}

		case "system":
			for _, line := range strings.Split(msg.content, "\n") {
				msgLines = append(msgLines, m.lip.Sys.Render(line))
			}
		}
		msgLines = append(msgLines, m.lip.SeparatorText)
	}

	if m.thinking {
		thinkLine := fmt.Sprintf("%s Thinking...", m.spinner.View())
		msgLines = append(msgLines, m.lip.Think.Render(thinkLine))
	}

	if m.err != "" {
		sanitized := strings.ReplaceAll(m.err, "\n", " ")
		msgLines = append(msgLines, m.lip.Err.Render(sanitized))
	}

	content := strings.Join(msgLines, "\n")
	m.viewport.SetContent(content)
	if m.needsAutoScroll {
		m.viewport.GotoBottom()
		m.needsAutoScroll = false
	}

	sidebarContent := m.renderSidebar(sidebarWidth)
	sidebar := m.lip.SidebarContainer.
		Width(sidebarWidth).
		MaxHeight(m.contentHeight).
		Render(sidebarContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, m.viewport.View(), sidebar)
	inputBox := m.textarea.View()
	if list := m.renderCompletionList(); list != "" {
		return lipgloss.JoinVertical(lipgloss.Top, topRow, inputBox, list)
	}
	return lipgloss.JoinVertical(lipgloss.Top, topRow, inputBox)
}

func (m *Model) renderSidebar(width int) string {
	var lines []string

	title := m.lip.SidebarActive.Render(" 状态")
	lines = append(lines, title)
	lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))

	if m.sidebar.Current != "" {
		spinner := ""
		if m.thinking {
			spinner = m.spinner.View() + " "
		}
		lines = append(lines, m.lip.SidebarActive.Render(spinner+m.sidebar.Current))
	}

	if conf.GetConfig().Cli.Tui.SidebarShowTokens && m.sidebar.Token != nil {
		t := m.sidebar.Token
		tokenLine := fmt.Sprintf("Token: ↑%d ↓%d Σ%d", t.PromptTokens, t.CompletionTokens, t.TotalTokens)
		lines = append(lines, m.lip.SidebarCompleted.Render(tokenLine))
		if t.CachedTokens > 0 {
			lines = append(lines, m.lip.SidebarCompleted.Render(fmt.Sprintf("  缓存: %d", t.CachedTokens)))
		}
		if t.ReasoningTokens > 0 {
			lines = append(lines, m.lip.SidebarCompleted.Render(fmt.Sprintf("  推理: %d", t.ReasoningTokens)))
		}
	}

	if len(m.sidebar.Log) > 0 || m.sidebar.Current != "" {
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
	}

	logContent := strings.Join(m.sidebar.Log, "\n")
	m.sidebarVP.SetContent(logContent)
	if m.thinking && len(m.sidebar.Log) > 0 {
		m.sidebarVP.GotoBottom()
	}
	logView := m.sidebarVP.View()
	if logView != "" {
		lines = append(lines, logView)
	}

	// 底部：工具确认选择 / 记忆选择 / 回退选择 / 占位
	// 确认的工具名和参数以 system 消息形式显示在主 viewport 中，此处仅显示选择列表
	lines = append(lines, "")
	if m.confirmingTool {
		lines = append(lines, m.lip.SidebarActive.Render(" 工具确认"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.confirmList.SetWidth(width - 4)
		lines = append(lines, m.confirmList.View())
	} else if m.selectingMem {
		lines = append(lines, m.lip.SidebarActive.Render(" 选择记忆"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.memoryList.SetWidth(width - 4)
		lines = append(lines, m.memoryList.View())
	} else if m.selectingReback {
		lines = append(lines, m.lip.SidebarActive.Render(" 选择回退"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.rebackList.SetWidth(width - 4)
		lines = append(lines, m.rebackList.View())
	} else {
		lines = append(lines, m.lip.SidebarPlaceholder.Render("(按 Tab 补全命令)"))
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderCompletionList() string {
	if !m.showingCompletions || len(m.completions) == 0 {
		return ""
	}
	maxVis := conf.GetConfig().Cli.Tui.CompletionMaxVisible
	if maxVis <= 0 {
		maxVis = 5
	}
	start := 0
	if m.completionIdx >= maxVis {
		start = m.completionIdx - maxVis + 1
	}
	end := start + maxVis
	if end > len(m.completions) {
		end = len(m.completions)
		start = end - maxVis
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.completionIdx {
			prefix = "> "
		}
		line := prefix + m.completions[i]
		if i == m.completionIdx {
			lines = append(lines, m.lip.SidebarActive.Render(line))
		} else {
			lines = append(lines, m.lip.SidebarCompleted.Render(line))
		}
	}
	list := strings.Join(lines, "\n")
	return m.lip.SidebarContainer.Render(list)
}
