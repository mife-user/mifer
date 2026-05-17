package lip

import "github.com/charmbracelet/bubbles/textarea"

func NewTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Enter 发送, Ctrl+N 换行, ↑↓ 历史)"
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.MaxHeight = 5 // 输入框最多 5 行，防止占用过多屏幕
	ta.Focus()
	return ta
}
