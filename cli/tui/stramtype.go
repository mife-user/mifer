package tui

// streamStatusMsg AI流式响应中的状态更新（agent切换、工具调用、工具错误）
type streamStatusMsg struct {
	event  string // "agent_start" | "agent_end" | "tool_start" | "tool_end" | "tool_error"
	name   string // agent名称或工具名称
	errMsg string // tool_error 时携带的错误消息
}

// streamContentMsg AI流式响应中的内容片段
type streamContentMsg struct {
	content string
}

// streamDoneMsg AI流式传输完成
type streamDoneMsg struct {
	err error
}
