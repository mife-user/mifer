package tui

import (
	"time"

	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
)

const (
	thinkingTickInterval = 500 * time.Millisecond
	contentMargin        = 4 // 消息区左右边距
	minHeight            = 10
	textareaLines        = 3
)

// message 对话消息
type message struct {
	role     string // "user" | "assistant" | "system"
	content  string // 原始内容
	rendered string // assistant 消息预渲染的 glamour 输出
}

// chatRespMsg SSE 全部积累完成后发送
type chatRespMsg struct {
	content string
	err     error
}

// thinkingTickMsg 思考动画 tick
type thinkingTickMsg struct{}

// systemMsg 系统命令结果
type systemMsg struct {
	content string
	err     error
}

// Model Bubble Tea 模型
type Model struct {
	client *client.Client
	config *conf.Config
	mark   *mark.Mark
	lip    *lip.Style

	messages    []message
	textarea    textarea.Model
	thinking    bool
	spinnerIdx  int
	scrollOff   int // 滚动偏移（从顶部算起的行数）
	lastMsgLine int // 上次渲染时的消息总行数，用于检测新消息
	err         string
	width       int
	height      int
}
