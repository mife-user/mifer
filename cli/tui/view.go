package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D787")).
			Bold(true)

	thinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Italic(true)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	sysStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD"))

	separator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Render(strings.Repeat("─", 60))
)

// View 渲染视图
func (m *Model) View() string {
	if m.width == 0 {
		return "正在启动..."
	}

	contentHeight := m.height - 5 // 保留 textarea 3行 + 边框 2行

	// 构建消息历史区域
	var msgLines []string
	// 从后往前收集，确保不超过可视区域
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := m.messages[i]
		var line string
		switch msg.role {
		case "user":
			line = userStyle.Render("You: ") + msg.content
		case "assistant":
			if msg.rendered != "" {
				line = msg.rendered
			} else {
				line = msg.content
			}
		case "system":
			line = sysStyle.Render(msg.content)
		}
		msgLines = append([]string{line}, msgLines...)
	}

	// thinking 指示器
	if m.thinking {
		spinner := spinnerFrames[m.spinnerIdx%len(spinnerFrames)]
		thinkLine := fmt.Sprintf("%s Thinking...", spinner)
		msgLines = append(msgLines, thinkStyle.Render(thinkLine))
	}

	// 错误信息
	if m.err != "" {
		msgLines = append(msgLines, errStyle.Render(m.err))
	}

	// 消息区域
	messageBox := strings.Join(msgLines, "\n" + separator + "\n")

	// 用 lipgloss 圆角边框包裹消息区域
	// 高度限制：只保留最后 contentHeight 行
	allLines := strings.Split(messageBox, "\n")
	if len(allLines) > contentHeight {
		allLines = allLines[len(allLines)-contentHeight:]
	}
	messageBox = strings.Join(allLines, "\n")

	// 确保消息区域填满可用高度
	for i := len(allLines); i < contentHeight; i++ {
		messageBox += "\n"
	}

	// 输入区域
	inputBox := m.textarea.View()

	return lipgloss.JoinVertical(lipgloss.Top, messageBox, inputBox)
}
