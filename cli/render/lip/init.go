package lip

import (
	"github.com/charmbracelet/lipgloss"
)

func Init() {
	lipgloss.NewStyle().Foreground(red).Bold(true)
}
