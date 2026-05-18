package tui

// systemMsg 由系统命令（/viewmemory、/excmem）的异步处理器发出。
// 与 chatRespMsg 的区别：systemMsg 的内容直接显示，不经过 markdown 渲染。
// err 非 nil 时显示错误信息，不追加消息到列表。
type systemMsg struct {
	content string
	err     error
}
