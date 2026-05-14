package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// View 渲染视图
func (m *Model) View() string {
	if m.width == 0 {
		return "正在启动..."
	}

	if m.height < minHeight {
		return fmt.Sprintf("窗口太小 (当前 %d, 最少 %d)", m.height, minHeight)
	}

	// 动态计算可用高度
	taHeight := m.height / 6
	if taHeight < 1 {
		taHeight = 1
	}
	// textarea 视图实际占用的行数 = taHeight 内容行
	contentHeight := m.height - taHeight - 1

	// 构建消息历史区域
	var msgLines []string
	for _, msg := range m.messages {
		var line string
		switch msg.role {
		case "user":
			line = m.lip.User.Render("You: ") + msg.content
		case "assistant":
			if msg.rendered != "" {
				line = msg.rendered
			} else {
				line = msg.content
			}
		case "system":
			line = m.lip.Sys.Render(msg.content)
		}
		// 将消息拆分为多行（glamour 渲染结果可能含 \n）
		msgLines = append(msgLines, strings.Split(line, "\n")...)
		msgLines = append(msgLines, m.lip.SeparatorText)
	}

	// thinking 指示器
	if m.thinking {
		spinner := spinnerFrames[m.spinnerIdx%len(spinnerFrames)]
		thinkLine := fmt.Sprintf("%s Thinking...", spinner)
		msgLines = append(msgLines, m.lip.Think.Render(thinkLine))
	}

	// 错误信息
	if m.err != "" {
		msgLines = append(msgLines, m.lip.Err.Render(m.err))
	}

	// 检测新消息 → 自动滚到底部
	if len(msgLines) != m.lastMsgLine {
		m.lastMsgLine = len(msgLines)
		if m.scrollOff > 0 {
			m.scrollOff = len(msgLines) - contentHeight // 滚到底部
		}
	}

	// 使用 scrollOffset 切片，而非截断
	maxOff := len(msgLines) - contentHeight
	if maxOff < 0 {
		maxOff = 0
	}
	if m.scrollOff > maxOff {
		m.scrollOff = maxOff
	}
	if m.scrollOff < 0 {
		m.scrollOff = 0
	}

	visible := msgLines
	if len(visible) > contentHeight {
		visible = visible[m.scrollOff : m.scrollOff+contentHeight]
	}

	// 显示滚动指示器
	if maxOff > 0 && m.scrollOff > 0 {
		visible = append([]string{m.lip.Scroll.Render(fmt.Sprintf("... 上方还有 %d 行 (滚轮查看)", m.scrollOff))}, visible...)
	}
	if maxOff > 0 && m.scrollOff < maxOff {
		visible = append(visible, m.lip.Scroll.Render(fmt.Sprintf("... 下方还有 %d 行", maxOff-m.scrollOff)))
	}

	// 消息区域
	messageBox := strings.Join(visible, "\n")

	// 确保消息区域填满可用高度
	for i := len(visible); i < contentHeight; i++ {
		messageBox += "\n"
	}

	// 输入区域
	inputBox := m.textarea.View()

	return lipgloss.JoinVertical(lipgloss.Top, messageBox, inputBox)
}
