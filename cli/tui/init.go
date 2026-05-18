package tui

// ============================================================================
// init.go — Model 初始化与 Bubble Tea 生命周期入口
// ============================================================================
//
// 外部调用流程（cli/cli.go → RunTUI）：
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

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel 创建 TUI 核心 Model，初始化所有子组件和依赖。
//
// 组件职责：
//
//	textarea (bubbles) → 用户输入区域，支持多行、占位符
//	viewport (bubbles) → 消息显示区域的滚动容器，处理鼠标滚轮和内容裁剪
//	spinner  (bubbles) → 等待 AI 响应时的旋转动画
//	mark    (glamour)  → 将 AI 返回的 markdown 文本渲染为终端 ANSI 彩色输出
//	lip     (lipgloss) → 预定义的消息样式集合（用户/AI/系统/错误等颜色）
func NewModel(client *client.Client, config *conf.Config) *Model {
	// ---- lip 样式：先于组件初始化，供 spinner 等使用 ----
	lipStyles := lip.Init(config)

	// ---- textarea：输入组件 ----
	ta := lip.NewTextarea()

	// ---- spinner：旋转动画组件 ----
	// 优先使用自定义帧，否则根据配置名称选择预置类型，均未设置时回退到 MiniDot
	sp := lip.NewSpinner(lipStyles, config)

	// ---- viewport：滚动视口组件 ----
	// 初始尺寸为 0，由第一个 WindowSizeMsg 事件设置正确尺寸
	vp := lip.NewViewport(config)

	// ---- memoryList：记忆选择列表组件 ----
	ml := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	ml.SetShowTitle(false)
	ml.SetShowStatusBar(false)
	ml.SetShowFilter(false)
	ml.SetShowPagination(false)
	ml.SetShowHelp(false)
	ml.SetFilteringEnabled(false)
	ml.DisableQuitKeybindings()

	// ---- memoryViewport：全屏记忆查看独立视口 ----
	mvp := lip.NewViewport(config)

	return &Model{
		// 依赖注入
		client: client,
		config: config,
		mark:   mark.Init(), // glamour 双渲染器（dark + notty 降级）
		lip:    lipStyles,   // lipgloss 样式集合

		// 消息与渲染
		messages: make([]message, 0),
		textarea: ta,
		spinner:  sp,
		viewport: vp,
		thinking: false,

		// 输入历史
		history:      make([]string, 0, config.Cli.Tui.MaxHistory),
		historyIdx:   -1, // -1 表示不在历史导航中
		pendingInput: "",

		// 侧边栏与流式传输
		sidebar:  SidebarState{},
		streamCh: nil,
		accBuf:   nil,

		// 记忆选择
		selectingMem: false,
		memoryList:   ml,

		// 全屏记忆查看
		showingMemoryView: false,
		memoryViewport:    mvp,
	}
}

// Init 是 Bubble Tea 生命周期的入口，返回初始命令。
//
// 这里返回 textarea.Blink 命令，让 textarea 的光标开始闪烁。
// Bubble Tea 框架会执行此命令，生成的 tea.Msg 进入 Update()。
//
// 注意：如果在此返回 nil，程序仍然正常运行，只是光标不会闪烁。
func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}
