package mark

import "mifer/pkg/logger"

func (m *Mark) Render(content string) (string, error) {
	rendered, err := m.Renderer.Render(content)
	if err != nil && m.Fallback != nil {
		// 主渲染器失败，降级到 notty 渲染（去除 markdown 标记符的纯文本）
		logger.Info("Markdown renderer failed, fallback to notty renderer")
		return m.Fallback.Render(content)
	}
	logger.Info("Markdown renderer success")
	return rendered, err
}
