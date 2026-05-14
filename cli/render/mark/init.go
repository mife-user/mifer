package mark

import "github.com/charmbracelet/glamour"

func Init() *Mark {
	// 主渲染器：使用默认 dark 风格（稳定可靠，不依赖终端检测）
	primary, err := glamour.NewTermRenderer(
		glamour.WithPreservedNewLines(),
		glamour.WithEmoji(),
	)
	if err != nil {
		panic(err)
	}

	// 降级渲染器：notty 风格会去除 markdown 标记符，输出纯文本
	fallback, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("notty"),
		glamour.WithPreservedNewLines(),
	)

	return &Mark{Renderer: primary, Fallback: fallback}
}
