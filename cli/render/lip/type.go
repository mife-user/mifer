package lip

import "github.com/charmbracelet/lipgloss"

type Style struct {
	base    *lipgloss.Style
	title   *lipgloss.Style
	content *lipgloss.Style
	err     *lipgloss.Style
	help    *lipgloss.Style
}
