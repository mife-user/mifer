package tui

import (
	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel 创建 TUI 模型
func NewModel(client *client.Client, config *conf.Config) *Model {
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Ctrl+C 退出)"
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.Focus()

	return &Model{
		client:   client,
		config:   config,
		mark:     mark.Init(),
		lip:      lip.Init(config),
		messages: make([]message, 0),
		textarea: ta,
		thinking: false,
	}
}

// Init 初始化命令
func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}
