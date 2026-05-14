package lip

import (
	"mifer/pkg/conf"

	"github.com/charmbracelet/lipgloss"
)

// Init 初始化所有 TUI 样式
// 从配置读取颜色，通过 Base.Inherit() 派生子样式，配置为空时使用硬编码降级颜色
func Init(config *conf.Config) Style {
	getFg := func(cfg string, fallback string) lipgloss.Color {
		if cfg != "" {
			return lipgloss.Color(cfg)
		}
		return lipgloss.Color(fallback)
	}

	// 基础样式：只设置公共属性（背景色、基础前景色、加粗）
	base := lipgloss.NewStyle().
		Background(lipgloss.Color(config.Cli.Lip.Base.Background)).
		Foreground(lipgloss.Color(config.Cli.Lip.Base.Foreground)).
		Bold(true)

	// 子样式从 Base 派生（lipgloss 方法返回新 style，无需显式 Copy）
	user := base.Foreground(getFg(config.Cli.Lip.Title.Foreground, "#00D787")).
		Bold(true)

	think := base.Foreground(getFg(config.Cli.Lip.Content.Foreground, "#FFB86C")).
		Bold(false).
		Italic(true)

	errStyle := base.Foreground(getFg(config.Cli.Lip.Err.Foreground, "#FF5555")).
		Bold(false)

	sys := base.Foreground(getFg(config.Cli.Lip.Help.Foreground, "#8BE9FD")).
		Bold(false)

	scroll := base.Foreground(lipgloss.Color("#666666")).
		Bold(false)

	separatorStyle := base.Foreground(lipgloss.Color("#444444")).
		Bold(false)

	sepText := separatorStyle.Render("────────────────────────────────────────────────────────────")

	return Style{
		Base:          &base,
		User:          &user,
		Think:         &think,
		Err:           &errStyle,
		Sys:           &sys,
		Scroll:        &scroll,
		Separator:     &separatorStyle,
		SeparatorText: sepText,
	}
}
