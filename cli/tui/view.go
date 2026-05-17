package tui

// ============================================================================
// view.go — Bubble Tea 渲染函数
// ============================================================================
//
// View() 在每次 Update() 返回后被 Bubble Tea 框架调用，生成终端 UI 字符串。
// 调用频率：每次事件处理后都会调用（按键、tick、窗口变化等）。
//
// 渲染管线（6 步流水线）：
//
//   ① 门控检查
//     - width == 0 → 程序尚未就绪，显示 "正在启动..."
//     - contentHeight < 1 → 窗口太小，提示最小高度
//
//   ② 构建消息行列表
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
	// 第 ① 步：门控检查
	// ======================================================================
	// 程序启动后首个 WindowSizeMsg 到达前 width 为 0，此时不渲染全部 UI
	if m.width == 0 {
		return "正在启动..."
	}
	// contentHeight < 1 表示窗口高度不足以显示消息区域
	if m.contentHeight < 1 {
		return fmt.Sprintf("窗口太小 (当前 %d, 最少 %d)", m.height, m.config.Cli.Tui.MinHeight)
	}

	// ======================================================================
	// 第 ② 步：构建消息行列表
	// ======================================================================
	// msgLines 是扁平的行切片，每行已应用对应的 lipgloss 样式
	var msgLines []string
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			// 用户消息：绿色粗体 "You: " 前缀 + 原始内容，单行渲染
			msgLines = append(msgLines, m.lip.User.Render("You: "+msg.content))

		case "assistant":
			// AI 消息：优先使用 glamour 预渲染的 ANSI 彩色输出
			// 预渲染字符串按 "\n" 拆分为多行，直接追加
			if msg.rendered != "" {
				msgLines = append(msgLines, strings.Split(msg.rendered, "\n")...)
			} else {
				// 降级：直接显示原始内容（始终会触发，因为 chatRespMsg
				// 处理中如果 mark.Render 失败会保留空 rendered）
				msgLines = append(msgLines, msg.content)
			}

		case "system":
			// 系统消息：青色渲染，支持多行（如记忆查看结果）
			for _, line := range strings.Split(msg.content, "\n") {
				msgLines = append(msgLines, m.lip.Sys.Render(line))
			}
		}
		// 每条消息后追加灰色分隔线（预渲染的字符串，避免每帧重复 Render）
		msgLines = append(msgLines, m.lip.SeparatorText)
	}

	// ======================================================================
	// 第 ③ 步：追加 thinking 旋转动画行
	// ======================================================================
	// m.thinking 在 handleEnter() 中设为 true，在 chatRespMsg 到达时设为 false
	// spinner.View() 返回当前帧的 braille 字符（由 tick 消息驱动帧切换）
	if m.thinking {
		thinkLine := fmt.Sprintf("%s Thinking...", m.spinner.View())
		msgLines = append(msgLines, m.lip.Think.Render(thinkLine))
	}

	// ======================================================================
	// 第 ④ 步：追加错误行
	// ======================================================================
	// err 在 chatRespMsg/systemMsg 解析失败时设置，在下一次发送消息时清空
	if m.err != "" {
		msgLines = append(msgLines, m.lip.Err.Render(m.err))
	}

	// ======================================================================
	// 第 ⑤ 步：设置 viewport 内容 + 自动滚底
	// ======================================================================
	// 将所有消息行拼接为单个字符串，设置到 viewport 中
	// viewport 内部按 "\n" 拆分为行数组，自动处理滚动边界
	content := strings.Join(msgLines, "\n")
	m.viewport.SetContent(content)

	// 仅在消息列表有新内容时才滚到底部
	// 如果用户手动向上滚动了视口，我们不强制拉回底部
	if m.needsAutoScroll {
		m.viewport.GotoBottom()
		m.needsAutoScroll = false
	}

	// ======================================================================
	// 第 ⑥ 步：组合输出
	// ======================================================================
	// viewport.View() 渲染消息区域（含背景色、圆角边框、内边距、滚动裁剪）
	// textarea.View() 渲染输入区域（占位符、光标等）
	// JoinVertical 将两者垂直拼接，Top 表示顶部对齐
	inputBox := m.textarea.View()
	return lipgloss.JoinVertical(lipgloss.Top, m.viewport.View(), inputBox)
}
