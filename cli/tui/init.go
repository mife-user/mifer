package tui

import (
	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func NewModel(client *client.Client) *Model {
	lipStyles := lip.Init()
	ta := lip.NewTextarea()
	sp := lip.NewSpinner(lipStyles)
	vp := lip.NewViewport()
	ml := lip.NewChoseList()
	rl := lip.NewChoseList()
	cl := lip.NewChoseList()
	pl := lip.NewChoseList()
	mvp := lip.NewViewport()
	svp := lip.NewViewport()

	return &Model{
		client: client,
		mark:   mark.Init(),
		lip:    lipStyles,

		messages: make([]message, 0),
		textarea: ta,
		spinner:  sp,
		viewport: vp,
		thinking: false,

		history:      make([]string, 0, conf.GetConfig().Cli.Tui.MaxHistory),
		historyIdx:   -1,
		pendingInput: "",

		sidebar:   SidebarState{},
		sidebarVP: svp,
		streamCh:  nil,
		accBuf:    nil,

		selectingMem: false,
		memoryList:   ml,

		selectingReback: false,
		rebackList:      rl,

		confirmingTool: false,
		confirmQueue:   make([]toolConfirmMsg, 0),
		confirmList:    cl,

		confirmingPlan: false,
		planList:       pl,

		showingMemoryView: false,
		memoryViewport:    mvp,
	}
}

func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}
