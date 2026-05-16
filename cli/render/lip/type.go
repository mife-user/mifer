package lip

import "github.com/charmbracelet/lipgloss"

// Style 聚合所有 TUI 使用的 lipgloss 样式。
//
// 样式继承关系：
//   Base（Bold + 可选前景色）→ 各子样式覆盖前景色
//
// 设计原则：
//   - 内联样式不设置 Background/Border（块级属性用于外层容器）
//   - 子样式仅设置前景色和粗体/斜体，不与 glamour markdown 渲染的 ANSI 码冲突
type Style struct {
	Base *lipgloss.Style // 基础样式：仅 Bold，不设置背景色和边框
	User *lipgloss.Style // 用户消息 "You: ..." 整行样式
	AI   *lipgloss.Style // AI 助手消息边框/前缀样式
	Think *lipgloss.Style // 思考动画 "⠋ Thinking..." 样式
	Err  *lipgloss.Style // 错误消息样式
	Sys  *lipgloss.Style // 系统消息样式
	Scroll *lipgloss.Style // 滚动指示器 "... N 行 ..." 样式
	Separator *lipgloss.Style // 分隔线样式
	SeparatorText string // 预渲染的分隔线文本
	Help *lipgloss.Style // 帮助/提示文本样式
}
