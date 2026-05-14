package lip

import (
	"mifer/pkg/conf"

	"github.com/charmbracelet/lipgloss"
)

func Init(config *conf.Config) Style {
	var style = lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.Cli.Lip.Base.Foreground)). // 前景颜色
		Background(lipgloss.Color(config.Cli.Lip.Base.Background)). // 背景颜色
		Bold(true).                                                 // 加粗
		Border(
			lipgloss.RoundedBorder(), // 圆角边框
		).
		BorderTop(true).                                                        // 上边框
		BorderLeft(true).                                                       // 左边框
		BorderRight(true).                                                      // 右边框
		BorderBottom(true).                                                     // 下边框
		BorderTopBackground(lipgloss.Color(config.Cli.Lip.Base.BoldTop)).       // 上边框背景颜色
		BorderLeftBackground(lipgloss.Color(config.Cli.Lip.Base.BoldLeft)).     // 左边框背景颜色
		BorderRightBackground(lipgloss.Color(config.Cli.Lip.Base.BoldRight)).   // 右边框背景颜色
		BorderBottomBackground(lipgloss.Color(config.Cli.Lip.Base.BoldBottom)). // 下边框背景颜色
		Padding(1, 2)                                                           // 内边距
	return Style{base: &style}
}
