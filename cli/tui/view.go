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
//   ④ 追加 thinking 旋转动画行
//     - 仅当 m.thinking == true 时渲染
//     - 格式：<spinner字符> Thinking...（橙色斜体）
//
//   ⑤ 追加错误行
//     - 仅当 m.err != "" 时渲染
//     - 红色文本，显示在消息列表末尾
//
//   ⑥ 设置 viewport 内容 + 自动滚底
//     - 将所有消息行用 "\n" 拼接为完整内容字符串
//     - 调用 viewport.SetContent(content) 设置行数据
//     - 如果 needsAutoScroll 标记为 true，调用 GotoBottom() 并重置标记
//
//   ⑦ 组合输出
//     - topRow = viewport.View() | 侧边栏（水平拼接）
//     - 如果输入以 / 开头，在 topRow 和 textarea 之间插入补全弹出栏
//     - 垂直拼接: topRow [, completionBar], textarea

import (
	"fmt"
	"strings"

	"mifer/pkg/conf"

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

		// 计划确认模式
		if m.showingPlanConfirm {
			title := m.lip.SidebarActive.Render(" 计划确认 — Enter 确认 / Esc 拒绝")
			sep := m.lip.SidebarSeparator.Render(strings.Repeat("─", m.width-4))
			return lipgloss.JoinVertical(lipgloss.Top, title, sep, m.planViewport.View())
		}

	// 全屏计划查看模式
	if m.showingPlanView {
		title := m.lip.SidebarActive.Render(" 计划查看 — Esc 返回")
		sep := m.lip.SidebarSeparator.Render(strings.Repeat("─", m.width-4))
		return lipgloss.JoinVertical(lipgloss.Top, title, sep, m.planViewport.View())
	}

	// ======================================================================
	// 第 ② 步：门控检查
	// ======================================================================
	if m.width == 0 {
		return "正在启动..."
	}
	if m.contentHeight < 1 {
		return fmt.Sprintf("窗口太小 (当前 %d, 最少 %d)", m.height, conf.GetConfig().Cli.Tui.MinHeight)
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
	// 第 ③ 步：构建消息行列表
	// ======================================================================
	var msgLines []string

	// 后端未就绪警告（api_key 未配置时显示）
	if m.backendWarning != "" {
		warnStyle := m.lip.Think.Bold(true)
		msgLines = append(msgLines, warnStyle.Render(m.backendWarning))
		msgLines = append(msgLines, m.lip.SidebarSeparator.Render(strings.Repeat("─", m.width-sidebarWidth-4)))
	}

	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			msgLines = append(msgLines, m.lip.User.Render("You: "+msg.content))
			msgLines = append(msgLines, m.lip.SeparatorText)

		case "assistant":
			msgLines = append(msgLines, m.lip.SeparatorText)
			if msg.rendered != "" {
				msgLines = append(msgLines, strings.Split(msg.rendered, "\n")...)
			} else {
				msgLines = append(msgLines, msg.content)
			}
			msgLines = append(msgLines, m.lip.SeparatorText)

		case "system":
			for _, line := range strings.Split(msg.content, "\n") {
				msgLines = append(msgLines, m.lip.Sys.Render(line))
			}
		}
	}

	// ======================================================================
	// 第 ④ 步：追加 thinking 内容（推理过程，灰色斜体，最多 3 行）
	//   布局：分隔线 ─→ thinking 内容 (≤3行) ─→ spinner 动画
	// ======================================================================
	if m.thinking && m.thinkingBuf != nil {
		content := strings.TrimRight(m.thinkingBuf.String(), "\n")
		if content != "" {
			wrapWidth := m.viewport.Width - 2 // 扣除 viewport 左右各 1 的 padding
			if wrapWidth < 20 {
				wrapWidth = 20
			}
			// 先按 \n 拆行，再对每行按 wrapWidth 硬换行
			rawLines := strings.Split(content, "\n")
			var allLines []string
			for _, line := range rawLines {
				allLines = append(allLines, wrapLine(line, wrapWidth)...)
			}
			// 取最后 3 行
			start := len(allLines) - 3
			if start < 0 {
				start = 0
			}
			// 分隔线
			msgLines = append(msgLines, m.lip.SidebarSeparator.Render(strings.Repeat("─", wrapWidth)))
			// thinking 内容
			for _, line := range allLines[start:] {
				msgLines = append(msgLines, m.lip.ThinkingText.Render(line))
			}
			// spinner 动画
			thinkLine := fmt.Sprintf("%s Thinking...", m.spinner.View())
			msgLines = append(msgLines, m.lip.Think.Render(thinkLine))
		}
	}

	// ======================================================================
	// 第 ⑤ 步：追加错误行
	// ======================================================================
	if m.err != "" {
		sanitized := strings.ReplaceAll(m.err, "\n", " ")
		msgLines = append(msgLines, m.lip.Err.Render(sanitized))
	}

	// ======================================================================
	// 第 ⑥ 步：设置 viewport 内容 + 自动滚底
	// ======================================================================
	content := strings.Join(msgLines, "\n")
	m.viewport.SetContent(content)
	if m.needsAutoScroll {
		m.viewport.GotoBottom()
		m.needsAutoScroll = false
	}

	// ======================================================================
	// 第 ⑦ 步：渲染侧边栏
	// ======================================================================
	sidebarContent := m.renderSidebar(sidebarWidth)
	sidebar := m.lip.SidebarContainer.
		Width(sidebarWidth).
		MaxHeight(m.contentHeight).
		Render(sidebarContent)

	// ======================================================================
	// 第 ⑧ 步：组合输出（水平：viewport | 侧边栏；垂直排列 textarea + [补全列表]）
	// ======================================================================
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, m.viewport.View(), sidebar)
	inputBox := m.textarea.View()
	if list := m.renderCompletionList(); list != "" {
		return lipgloss.JoinVertical(lipgloss.Top, topRow, inputBox, list)
	}
	return lipgloss.JoinVertical(lipgloss.Top, topRow, inputBox)
}

// renderSidebar 渲染右侧状态侧边栏
func (m *Model) renderSidebar(width int) string {
	var lines []string

	// 标题行
	title := m.lip.SidebarActive.Render(" 状态")
	lines = append(lines, title)
	lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))

	// 当前活跃项（带 spinner 动画）
	if m.sidebar.Current != "" {
		spinner := ""
		if m.thinking {
			spinner = m.spinner.View() + " "
		}
		lines = append(lines, m.lip.SidebarActive.Render(spinner+m.sidebar.Current))
	}

	// Token 统计行（可配置）
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

	// 分隔线
	if len(m.sidebar.Log) > 0 || m.sidebar.Current != "" {
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
	}

	// 事件日志（viewport 滚动区域）
	logContent := strings.Join(m.sidebar.Log, "\n")
	m.sidebarVP.SetContent(logContent)
	if m.thinking && len(m.sidebar.Log) > 0 {
		m.sidebarVP.GotoBottom()
	}
	logView := m.sidebarVP.View()
	if logView != "" {
		lines = append(lines, logView)
	}

	// 底部：记忆选择列表或占位
	lines = append(lines, "")
	if m.selectingMem {
		lines = append(lines, m.lip.SidebarActive.Render(" 选择记忆"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.memoryList.SetWidth(width - 4)
		lines = append(lines, m.memoryList.View())
	} else if m.selectingReback {
		lines = append(lines, m.lip.SidebarActive.Render(" 选择回退"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.rebackList.SetWidth(width - 4)
		lines = append(lines, m.rebackList.View())
	} else if m.selectingPlan {
		lines = append(lines, m.lip.SidebarActive.Render(" 选择计划"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.planList.SetWidth(width - 4)
		lines = append(lines, m.planList.View())
	} else if m.selectingConfirm {
		lines = append(lines, m.lip.SidebarActive.Render(" 确认执行↑↓选择"))
		lines = append(lines, m.lip.SidebarSeparator.Render(strings.Repeat("─", width-3)))
		m.confirmList.SetWidth(width - 4)
		lines = append(lines, m.confirmList.View())
	} else {
		lines = append(lines, m.lip.SidebarPlaceholder.Render("(按 Tab 补全命令)"))
	}

	return strings.Join(lines, "\n")
}

// renderCompletionList 渲染斜杠命令补全列表（输入框下方）
// 仅在 showingCompletions && 存在匹配命令时显示
// 最多显示 CompletionMaxVisible 条，超出部分基于 completionIdx 滚动视窗
func (m *Model) renderCompletionList() string {
	if !m.showingCompletions || len(m.completions) == 0 {
		return ""
	}
	maxVis := conf.GetConfig().Cli.Tui.CompletionMaxVisible
	if maxVis <= 0 {
		maxVis = 5
	}
	// 计算可见窗口
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
		cmdName := m.completions[i]
		line := prefix + cmdName
		// 如果有描述，追加在命令后面
		if desc, ok := m.completionDescs[cmdName]; ok && desc != "" {
			line += "  —  " + desc
		}
		if i == m.completionIdx {
			lines = append(lines, m.lip.SidebarActive.Render(line))
		} else {
			lines = append(lines, m.lip.SidebarCompleted.Render(line))
		}
	}
	list := strings.Join(lines, "\n")
	return m.lip.SidebarContainer.Render(list)
}

// ============================================================================
// 辅助函数
// ============================================================================

// wrapLine 按指定宽度硬换行（字符级），用于 thinking 内容显示。
// 参数 maxWidth 为 rune 宽度，<=0 时返回原行。
func wrapLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}
	var result []string
	runes := []rune(line)
	for len(runes) > maxWidth {
		result = append(result, string(runes[:maxWidth]))
		runes = runes[maxWidth:]
	}
	if len(runes) > 0 {
		result = append(result, string(runes))
	}
	return result
}
