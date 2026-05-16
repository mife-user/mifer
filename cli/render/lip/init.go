package lip

import (
	"mifer/pkg/conf"

	"github.com/charmbracelet/lipgloss"
)

// Init 初始化所有 TUI 样式。
//
// 样式继承链：
//   base (Bold, 无背景/边框) → 各子样式覆盖前景色
//
// 设计决策：
//   - 内联文本不设 Background/Border，避免与 glamour markdown 的 ANSI 码冲突
//   - 背景色在外层容器（View 的消息区域）统一应用
//   - 配置值为空时使用硬编码降级颜色，确保终端兼容
func Init(config *conf.Config) *Style {
	// 辅助函数：配置为空时回退到硬编码默认色
	getFg := func(cfg string, fallback string) lipgloss.Color {
		if cfg != "" {
			return lipgloss.Color(cfg)
		}
		return lipgloss.Color(fallback)
	}

	// 基础样式：仅 Bold，不设 Background/Border（此二者由外层容器统一处理）
	base := lipgloss.NewStyle().
		Bold(true)

	// ---- 子样式：各消息类型覆盖前景色 ----
	user := base.
		Foreground(getFg(config.Cli.Lip.Title.Foreground, "#00D787")).
		Bold(true)

	ai := base.
		Foreground(getFg(config.Cli.Lip.Base.Foreground, "#00ff11")).
		Bold(false)

	think := base.
		Foreground(getFg(config.Cli.Lip.Content.Foreground, "#FFB86C")).
		Bold(false).
		Italic(true)

	errStyle := base.
		Foreground(getFg(config.Cli.Lip.Err.Foreground, "#FF5555")).
		Bold(false)

	sys := base.
		Foreground(getFg(config.Cli.Lip.Help.Foreground, "#8BE9FD")).
		Bold(false)

	scroll := base.
		Foreground(lipgloss.Color("#666666")).
		Bold(false)

	separatorStyle := base.
		Foreground(lipgloss.Color("#444444")).
		Bold(false)

	sepText := separatorStyle.Render("────────────────────────────────────────────────────────────")

	help := base.
		Foreground(lipgloss.Color("#888888")).
		Bold(false)

	return &Style{
		Base:          &base,
		User:          &user,
		AI:            &ai,
		Think:         &think,
		Err:           &errStyle,
		Sys:           &sys,
		Scroll:        &scroll,
		Separator:     &separatorStyle,
		SeparatorText: sepText,
		Help:          &help,
	}
}
