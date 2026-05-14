package tui

import (
	"time"

	"mifer/cli/client"
	"mifer/cli/render/lip"
	"mifer/cli/render/mark"
	"mifer/pkg/conf"

	"github.com/charmbracelet/bubbles/textarea"
)

const thinkingTickInterval = 500 * time.Millisecond

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
	lip    lip.Style

	messages   []message
	textarea   textarea.Model // 输入文本框
	thinking   bool
	spinnerIdx int // 思考动画索引
	err        string
	width      int // 窗口宽度
	height     int // 窗口高度
}
