package agentres

// MCPServerStatus MCP Server 状态
type MCPServerStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`     // connected / disabled / error / disconnected
	ToolCount int    `json:"tool_count"` // 已加载的工具数量
	Error     string `json:"error,omitempty"`
}

// MCPStatusRes MCP 状态响应
type MCPStatusRes struct {
	Servers []MCPServerStatus `json:"servers"`
}
