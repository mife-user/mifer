package tui

// SidebarState 右侧边栏状态，跟踪当前agent/工具执行情况
type SidebarState struct {
	CurrentAgent string   // 当前正在执行的Agent名称
	CurrentTool  string   // 当前正在调用的工具名称
	AgentTrail   []string // 已完成Agent的轨迹
	ToolTrail    []string // 已完成工具的轨迹
	ToolError    string   // 最近一次工具调用的错误消息（为空表示无错误）
}

// update 根据流式状态消息更新侧边栏状态
func (s *SidebarState) update(msg streamStatusMsg) {
	switch msg.event {
	case "agent_start":
		if s.CurrentAgent != "" {
			s.AgentTrail = append(s.AgentTrail, s.CurrentAgent)
		}
		s.CurrentAgent = msg.name
		s.CurrentTool = "" // 新agent开始时重置当前工具
	case "agent_end":
		if s.CurrentAgent == msg.name {
			s.AgentTrail = append(s.AgentTrail, s.CurrentAgent)
			s.CurrentAgent = ""
		}
	case "tool_start":
		if s.CurrentTool != "" {
			s.ToolTrail = append(s.ToolTrail, "  "+s.CurrentTool)
		}
		s.CurrentTool = msg.name
		s.ToolError = "" // 新工具启动时清空上一次错误
	case "tool_end":
		if s.CurrentTool == msg.name {
			trail := "  " + s.CurrentTool
			if s.ToolError != "" {
				trail += " [ERROR]"
			}
			s.ToolTrail = append(s.ToolTrail, trail)
			s.CurrentTool = ""
		}
	case "tool_error":
		s.ToolError = msg.errMsg
	}
}
