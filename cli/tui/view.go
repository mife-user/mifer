package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// 旋转动画帧（braille 字符，10 帧循环）
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ============================================================================
// View — Bubble Tea 渲染入口
// ============================================================================
// 渲染管线（从上到下）：
//   1. 门控检查（未就绪 / 窗口太小）
//   2. 构建消息行列表（遍历 messages → 角色前缀 + 内容 + 分隔线）
//   3. 追加 thinking 旋转动画行（如 m.thinking = true）
//   4. 追加错误行（如 m.err != ""）
//   5. 滚动切片（用 m.scrollOff 裁切可见行）
//   6. 滚动指示器（上下方剩余行数提示）
//   7. 填充消息区域到 contentHeight 行
//   8. 外层容器应用背景色
//   9. JoinVertical(消息区域, 输入区域)

func (m *Model) View() string {
	// ---- 第 1 步：门控 ----
	if m.width == 0 {
		return "正在启动..."
	}
	if m.contentHeight < 1 {
		return fmt.Sprintf("窗口太小 (当前 %d, 最少 %d)", m.height, m.config.Cli.Tui.MinHeight)
	}

	// ---- 第 2 步：构建消息行列表 ----
	var msgLines []string
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			// 用户消息：绿色 "You: " 前缀 + 原始内容，整行统一渲染
			msgLines = append(msgLines, m.lip.User.Render("You: "+msg.content))

		case "assistant":
			// AI 消息：优先用 glamour 预渲染的 ANSI 输出
			if msg.rendered != "" {
				msgLines = append(msgLines, strings.Split(msg.rendered, "\n")...)
			} else {
				msgLines = append(msgLines, msg.content)
			}

		case "system":
			// 系统消息：青色整行渲染
			for _, line := range strings.Split(msg.content, "\n") {
				msgLines = append(msgLines, m.lip.Sys.Render(line))
			}
		}
		// 每条消息后追加分隔线
		msgLines = append(msgLines, m.lip.SeparatorText)
	}

	// ---- 第 3 步：thinking 旋转动画 ----
	if m.thinking {
		spinner := spinnerFrames[m.spinnerIdx%len(spinnerFrames)]
		thinkLine := fmt.Sprintf("%s Thinking...", spinner)
		msgLines = append(msgLines, m.lip.Think.Render(thinkLine))
	}

	// ---- 第 4 步：错误行 ----
	if m.err != "" {
		msgLines = append(msgLines, m.lip.Err.Render(m.err))
	}

	// ---- 第 5 步：滚动切片 ----
	// 限制 scrollOff 范围
	maxOff := max(len(msgLines)-m.contentHeight, 0)
	if m.scrollOff > maxOff {
		m.scrollOff = maxOff
	}
	if m.scrollOff < 0 {
		m.scrollOff = 0
	}

	visible := msgLines
	if len(visible) > m.contentHeight {
		visible = visible[m.scrollOff : m.scrollOff+m.contentHeight]
	}

	// ---- 第 6 步：滚动指示器 ----
	if maxOff > 0 && m.scrollOff > 0 {
		indicator := m.lip.Scroll.Render(fmt.Sprintf("... 上方还有 %d 行 (滚轮查看)", m.scrollOff))
		visible = append([]string{indicator}, visible...)
	}
	if maxOff > 0 && m.scrollOff < maxOff {
		indicator := m.lip.Scroll.Render(fmt.Sprintf("... 下方还有 %d 行", maxOff-m.scrollOff))
		visible = append(visible, indicator)
	}

	// ---- 第 7 步：填充到 contentHeight 行 ----
	for len(visible) < m.contentHeight {
		visible = append(visible, "")
	}
	messageBox := strings.Join(visible, "\n")

	// ---- 第 8 步：外层容器应用背景色 ----
	// 背景色仅在外层容器设置，避免与内联文本和 glamour ANSI 码冲突
	bgColor := m.config.Cli.Lip.Base.Background
	if bgColor == "" {
		bgColor = "#1e1e1e" // 硬编码降级
	}
	messageBox = lipgloss.NewStyle().
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1). // 左右各 1 列留白
		Render(messageBox)

	// ---- 第 9 步：组合输出 ----
	inputBox := m.textarea.View()
	return lipgloss.JoinVertical(lipgloss.Top, messageBox, inputBox)
}
