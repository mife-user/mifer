package mark

import "github.com/charmbracelet/glamour"

// Init 初始化 markdown 渲染器（双渲染器策略）。
//
// 主渲染器：glamour 默认 dark 风格 —— 输出带 ANSI 颜色码的终端文本
// 降级渲染器：notty 风格 —— 去除所有 markdown 标记符，输出纯文本
//
// notty 降级器在终端不支持 dark 主题的 ANSI 转义序列时使用，
// 确保无论如何都能产生可读输出（而非乱码或空白）。
func Init() *Mark {
	primary, err := glamour.NewTermRenderer(
		glamour.WithPreservedNewLines(),
		glamour.WithEmoji(),
	)
	if err != nil {
		panic(err)
	}

	fallback, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("notty"),
		glamour.WithPreservedNewLines(),
	)

	return &Mark{Renderer: primary, Fallback: fallback}
}
