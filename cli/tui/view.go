package tui

// ============================================================================
// view.go — Bubble Tea 渲染函数
// ============================================================================
//
// View() 在每次 Update() 返回后被 Bubble Tea 框架调用，生成终端 UI 字符串。
// 调用频率：每次事件处理后都会调用（按键、tick、窗口变化等）。
//
// 渲染管线（7 步流水线）：
//
//   ① 全屏记忆查看模式
//     - showingMemoryView == true → 独立 viewport + 标题 "对话记忆 — Esc 返回"
//     - 不显示侧边栏、textarea
//
//   ② 门控检查
//     - width == 0 → 程序尚未就绪，显示 "正在启动..."
//     - contentHeight < 1 → 窗口太小，提示最小高度
//
//   ③ 构建消息行列表
//     - 遍历 m.messages，按 role 分发渲染
//       · user      → "You: " + 内容，绿色粗体
//       · assistant → glamour 预渲染的 ANSI 字符串（拆行）
//       · system    → 青色渲染，支持多行内容
//     - 每条消息后追加分隔线（灰色虚线）
//
//   ③ 追加 thinking 旋转动画行
//     - 仅当 m.thinking == true 时渲染
//     - 格式：<spinner字符> Thinking...（橙色斜体）
//
//   ④ 追加错误行
//     - 仅当 m.err != "" 时渲染
//     - 红色文本，显示在消息列表末尾
//
//   ⑤ 设置 viewport 内容 + 自动滚底
//     - 将所有消息行用 "\n" 拼接为完整内容字符串
//     - 调用 viewport.SetContent(content) 设置行数据
//     - 如果 needsAutoScroll 标记为 true，调用 GotoBottom() 并重置标记
//
//   ⑥ 组合输出
//     - viewport.View() → 消息区域（含背景色、边框、内边距、滚动裁剪）
//     - textarea.View() → 输入区域
//     - lipgloss.JoinVertical(Top, viewport, textarea) → 垂直拼接

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	// ======================================================================
	// 第 ① 步：全屏记忆查看模式 — 独立 viewport，无其他 UI 元素
	// ======================================================================
	if m.showingMemoryView {
		title := m.lip.SidebarActive.Render(" 对话记忆 — Esc 返回")
		sep := m.lip.SidebarSeparator.Render(strings.Repeat("─", m.width-4))
		return lipgloss.JoinVertical(lipgloss.Top, title, sep, m.memoryViewport.View())
	}

	// ======================================================================
	// 第 ② 步：门控检查
	// ======================================================================
	if m.width == 0 {
		return "正在启动..."
	}
	if m.contentHeight < 1 {
		return fmt.Sprintf("窗口太小 (当前 %d, 最少 %d)", m.height, m.config.Cli.Tui.MinHeight)
	}

	// 计算侧边栏宽度：终端宽度 1/4，限制在 [20, 40]
	sidebarWidth := m.width / 4
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}

	// ======================================================================
	// 第 ② 步：构建消息行列表
	// ======================================================================
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

	// ======================================================================
	// 第 ③ 步：追加 thinking 旋转动画行
	// ======================================================================
	if m.thinking {
		thinkLine := fmt.Sprintf("%s Thinking...", m.spinner.View())
		msgLines = append(msgLines, m.lip.Think.Render(thinkLine))
	}

	// ======================================================================
	// 第 ④ 步：追加错误行
	// ======================================================================
	if m.err != "" {
		sanitized := strings.ReplaceAll(m.err, "\n", " ")
		msgLines = append(msgLines, m.lip.Err.Render(sanitized))
	}

	// ======================================================================
	// 第 ⑤ 步：设置 viewport 内容 + 自动滚底
	// ======================================================================
	content := strings.Join(msgLines, "\n")
	m.viewport.SetContent(content)
	if m.needsAutoScroll {
		m.viewport.GotoBottom()
		m.needsAutoScroll = false
	}

	// ======================================================================
	// 第 ⑥ 步：渲染侧边栏
	// ======================================================================
	sidebarContent := m.renderSidebar(sidebarWidth)
	sidebar := m.lip.SidebarContainer.
		Width(sidebarWidth).
		MaxHeight(m.contentHeight).
		Render(sidebarContent)

	// ======================================================================
	// 第 ⑦ 步：组合输出（水平：viewport | 侧边栏；垂直排列 textarea）
	// ======================================================================
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, m.viewport.View(), sidebar)
	inputBox := m.textarea.View()
	return lipgloss.JoinVertical(lipgloss.Top, topRow, inputBox)
}

// renderSidebar 渲染右侧边栏内容
func (m *Model) renderSidebar(width int) string {
	var lines []string

	// 标题行
	title := m.lip.SidebarActive.Render(" Agent / Tool")
	lines = append(lines, title)
	lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))

	// 当前活跃 Agent（带 spinner 动画）
	if m.sidebar.CurrentAgent != "" {
		spinner := ""
		if m.thinking {
			spinner = m.spinner.View() + " "
		}
		lines = append(lines, m.lip.SidebarActive.Render(spinner+m.sidebar.CurrentAgent))
	}

	// 当前活跃工具（缩进显示，表示隶属于当前 Agent）
	if m.sidebar.CurrentTool != "" {
		spinner := ""
		if m.thinking {
			spinner = m.spinner.View() + " "
		}
		lines = append(lines, m.lip.SidebarActive.Render("  "+spinner+m.sidebar.CurrentTool))
		// 活跃工具出错时显示错误消息
		if m.sidebar.ToolError != "" {
			lines = append(lines, m.lip.Err.Render("  E: "+m.sidebar.ToolError))
		}
	}

	// 空行分隔活跃项与已完成轨迹
	if len(m.sidebar.AgentTrail) > 0 || len(m.sidebar.ToolTrail) > 0 {
		lines = append(lines, "")
	}

	// 已完成 Agent 轨迹（灰色，最多 5 条）
	agentStart := 0
	if len(m.sidebar.AgentTrail) > 5 {
		agentStart = len(m.sidebar.AgentTrail) - 5
	}
	for _, a := range m.sidebar.AgentTrail[agentStart:] {
		lines = append(lines, m.lip.SidebarCompleted.Render(a))
	}

	// 已完成工具轨迹（灰色，缩进，最多 5 条）
	toolStart := 0
	if len(m.sidebar.ToolTrail) > 5 {
		toolStart = len(m.sidebar.ToolTrail) - 5
	}
	for _, t := range m.sidebar.ToolTrail[toolStart:] {
		lines = append(lines, m.lip.SidebarCompleted.Render(t))
	}

	// 底部：记忆选择列表或占位
	lines = append(lines, "")
	if m.selectingMem {
		lines = append(lines, m.lip.SidebarActive.Render(" 选择记忆"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		// 设置 list 宽度（减去容器内边距和边框）
		m.memoryList.SetWidth(width - 4)
		lines = append(lines, m.memoryList.View())
	} else {
		lines = append(lines, m.lip.SidebarPlaceholder.Render("(代码预览)"))
	}

	content := strings.Join(lines, "\n")
	return content
}
