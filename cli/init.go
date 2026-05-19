package cli

import (
	"fmt"
	"mifer/cli/client"
	"mifer/cli/tui"
	"mifer/pkg/conf"

	tea "github.com/charmbracelet/bubbletea"
)

// Cli TUI 交互客户端
type Cli struct {
	client *client.Client
	config *conf.Config
}

// New 创建 CLI 实例
func New(config *conf.Config) *Cli {
	baseURL := fmt.Sprintf("http://%s:%d", config.Cli.Host, config.Cli.Port)
	return &Cli{
		client: client.New(baseURL),
		config: config,
	}
}

// Run 运行 Bubble Tea TUI 交互模式
func (c *Cli) Run() error {
	m := tui.NewModel(c.client, c.config)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
