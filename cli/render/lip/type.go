package lip

import "github.com/charmbracelet/lipgloss"

// Style 聚合所有 TUI 使用的 lipgloss 样式
// 子样式通过 Base.Inherit() 派生，继承背景色等公共属性，覆盖各自的前景色
type Style struct {
	Base          *lipgloss.Style // 基础样式（前景色、背景色、加粗）
	User          *lipgloss.Style // 用户消息样式（配置: Title.Foreground）
	Think         *lipgloss.Style // 思考动画样式（配置: Content.Foreground）
	Err           *lipgloss.Style // 错误消息样式（配置: Err.Foreground）
	Sys           *lipgloss.Style // 系统消息样式（配置: Help.Foreground）
	Scroll        *lipgloss.Style // 滚动指示器样式
	Separator     *lipgloss.Style // 分隔线样式
	SeparatorText string         // 预渲染分隔线文本
}
