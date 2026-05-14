package mark

import "github.com/charmbracelet/glamour"

func Init() *Mark {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithEmoji(),
	)
	if err != nil {
		panic(err)
	}
	return &Mark{
		Renderer: r,
	}
}
