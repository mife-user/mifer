package lip

import (
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

func NewViewport(config *conf.Config) viewport.Model {
	vp := viewport.New(0, 0)
	vp.MouseWheelDelta = 1 // 滚轮每次滚动 1 行
	// 视口样式：沿用原来外层容器的背景色、内边距、圆角边框
	bgColor := config.Cli.Lip.Base.Background
	if bgColor == "" {
		bgColor = "#1e1e1e" // 配置为空时降级到深色背景
	}
	vp.Style = lipgloss.NewStyle().
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1).                   // 左右各 1 列留白
		Border(lipgloss.RoundedBorder()) // 圆角边框
	return vp
}
