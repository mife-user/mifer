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

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	// ---- textarea：输入组件 ----
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Enter 发送, Ctrl+N 换行, ↑↓ 历史)"
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.MaxHeight = 5 // 输入框最多 5 行，防止占用过多屏幕
	ta.Focus()

	// ---- spinner：旋转动画组件 ----
	// MiniDot 使用 braille 点阵字符（⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏），10 帧循环
	// FPS 约 12（~83ms/帧），视觉效果平滑
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))

	// ---- viewport：滚动视口组件 ----
	// 初始尺寸为 0，由第一个 WindowSizeMsg 事件设置正确尺寸
	vp := viewport.New(0, 0)
	vp.MouseWheelDelta = 1 // 滚轮每次滚动 1 行
	// 视口样式：沿用原来外层容器的背景色、内边距、圆角边框
	bgColor := config.Cli.Lip.Base.Background
	if bgColor == "" {
		bgColor = "#1e1e1e" // 配置为空时降级到深色背景
	}
	vp.Style = lipgloss.NewStyle().
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1).                   // 左右各 1 列留白
		Border(lipgloss.RoundedBorder()) // 圆角边框

	return &Model{
		// 依赖注入
		client: client,
		config: config,
		mark:   mark.Init(),      // glamour 双渲染器（dark + notty 降级）
		lip:    lip.Init(config), // lipgloss 样式集合

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
