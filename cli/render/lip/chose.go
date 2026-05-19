package lip

import "github.com/charmbracelet/bubbles/list"

func NewChoseList() list.Model {
	ml := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	ml.SetShowTitle(false)
	ml.SetShowStatusBar(false)
	ml.SetShowFilter(false)
	ml.SetShowPagination(false)
	ml.SetShowHelp(false)
	ml.SetFilteringEnabled(false)
	ml.DisableQuitKeybindings()
	return ml
}
