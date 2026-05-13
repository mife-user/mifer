package mark

import "github.com/charmbracelet/glamour"

func Init() {
	glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
}
