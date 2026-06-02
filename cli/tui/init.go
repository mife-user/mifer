package tui

// ============================================================================
// init.go — Model 初始化与 Bubble Tea 生命周期入口
// ============================================================================
//
// 外部调用流程（cli/init.go → Run）：
//
//   m := tui.NewModel(client, config)            // ① 创建 Model
//   p := tea.NewProgram(m, WithAltScreen(), ...)  // ② 创建 Bubble Tea 程序
//   p.Run()                                       // ③ 进入事件循环
//
// Bubble Tea 事件循环内部流程：
//   1. 调用 Model.Init() → 获取初始命令（如 textarea.Blink）
//   2. 执行命令 → 产生的消息送入 Update()
//   3. Update(msg) → 更新状态 → 返回新命令
//   4. View() → 渲染 UI 字符串 → 输出到终端
//   5. 重复 2-4 直到 tea.Quit 命令

import (
	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel 创建 TUI 核心 Model，初始化所有子组件和依赖。
func NewModel(client *client.Client) *Model {
	lipStyles := lip.Init()
	ta := lip.NewTextarea()
	sp := lip.NewSpinner(lipStyles)
	vp := lip.NewViewport()
	ml := lip.NewChoseList()
	rl := lip.NewChoseList()
	pl := lip.NewChoseList()
	cl := lip.NewChoseList()
	mvp := lip.NewViewport()
	pvp := lip.NewViewport()
	svp := lip.NewViewport()

	return &Model{
		// 依赖注入
		client: client,
		mark:   mark.Init(),
		lip:    lipStyles,

		// 消息与渲染
		messages: make([]message, 0),
		textarea: ta,
		spinner:  sp,
		viewport: vp,
		thinking: false,

		// 输入历史
		history:      make([]string, 0, conf.GetConfig().Cli.Tui.MaxHistory),
		historyIdx:   -1,
		pendingInput: "",

		// 侧边栏与流式传输
		sidebar:   SidebarState{},
		sidebarVP: svp,
		streamCh:  nil,
		accBuf:    nil,

		// 记忆选择
		selectingMem: false,
		memoryList:   ml,

		// 回退选择
		selectingReback: false,
		rebackList:      rl,

		// 全屏记忆查看
		showingMemoryView: false,
		memoryViewport:    mvp,

		// 计划选择
		selectingPlan: false,
		planList:      pl,

		// 全屏计划查看
		showingPlanView: false,
		planViewport:    pvp,

		// 工具确认
		selectingConfirm: false,
		confirmQueue:     nil,
		confirmList:      cl,
		sessionAllowed:   make(map[string]bool),
	}
}

// Init Bubble Tea 生命周期入口
func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}
