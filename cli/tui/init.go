package tui

import (
	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel 创建 TUI 核心 Model，初始化所有子组件。
//
// 初始化顺序：
//  1. textarea — 输入组件（默认 1 行，Enter 提交，Ctrl+Enter 换行）
//  2. mark — glamour markdown 渲染器（dark 主题 + notty 降级）
//  3. lip — lipgloss 样式集合（前景色、分隔线等）
func NewModel(client *client.Client, config *conf.Config) *Model {
	// 输入区域：默认 1 行高，内容超出宽度自动换行
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Enter 发送, Ctrl+N 换行, ↑↓ 历史)"
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.MaxHeight = 5 // 输入框最多 5 行，防止占用过多屏幕
	ta.Focus()

	return &Model{
		client:       client,
		config:       config,
		mark:         mark.Init(),
		lip:          lip.Init(config),
		messages:     make([]message, 0),
		textarea:     ta,
		thinking:     false,
		history:      make([]string, 0, config.Cli.Tui.MaxHistory),
		historyIdx:   -1,
		pendingInput: "",
	}
}

// Init 返回初始命令：光标闪烁动画
func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}
