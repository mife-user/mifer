package mark

import "github.com/charmbracelet/glamour"

type Mark struct {
	Renderer *glamour.TermRenderer // 主渲染器（dark 风格）
	Fallback *glamour.TermRenderer // 降级渲染器（notty 风格，去除 markdown 标记符）
}
