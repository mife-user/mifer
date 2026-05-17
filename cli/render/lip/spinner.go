package lip

import (
	"mifer/pkg/conf"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

func NewSpinner(lipStyles *Style, config *conf.Config) spinner.Model {
	sp := spinner.New(
		spinner.WithSpinner(pickSpinner(config)),
		spinner.WithStyle(*lipStyles.Think),
	)
	return sp
}

// pickSpinner 根据配置返回 spinner.Spinner：自定义帧优先，其次预置类型名，最后回退 MiniDot。
func pickSpinner(config *conf.Config) spinner.Spinner {
	tui := config.Cli.Tui
	// 自定义帧优先
	if len(tui.SpinnerFrames) > 0 {
		fps := tui.SpinnerFPS
		if fps <= 0 {
			fps = 10
		}
		return spinner.Spinner{
			Frames: tui.SpinnerFrames,
			FPS:    time.Second / time.Duration(fps),
		}
	}
	// 预置类型
	switch tui.SpinnerType {
	case "Line":
		return spinner.Line
	case "Dot":
		return spinner.Dot
	case "Jump":
		return spinner.Jump
	case "Pulse":
		return spinner.Pulse
	case "Points":
		return spinner.Points
	case "Globe":
		return spinner.Globe
	case "Moon":
		return spinner.Moon
	case "Monkey":
		return spinner.Monkey
	default:
		return spinner.MiniDot
	}
}
