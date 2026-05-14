package mark

func (m *Mark) Render(content string) (string, error) {
	return m.Renderer.Render(content)
}
