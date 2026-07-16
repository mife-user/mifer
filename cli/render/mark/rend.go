package mark

import (
	"context"

	"mifer/pkg/logger"
)

// Render 将 markdown 文本渲染为终端 ANSI 字符串。
//
// 渲染策略：
//   1. 主渲染器（dark 主题）→ 输出带 ANSI 颜色码的终端文本
//   2. 失败时降级到 notty 渲染器 → 去除 markdown 标记符的纯文本
//
// 降级原因：部分终端不支持 glamour dark 主题的 ANSI 序列，
// 此时用 notty 风格确保内容可读，避免渲染失败导致空白输出。
func (m *Mark) Render(content string) (string, error) {
	rendered, err := m.Renderer.Render(content)
	if err != nil && m.Fallback != nil {
		logger.Info(context.Background(), "Markdown renderer failed, fallback to notty renderer")
		return m.Fallback.Render(content)
	}
	logger.Info(context.Background(), "Markdown renderer success")
	return rendered, err
}
