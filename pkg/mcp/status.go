package mcp

// ServerStatus 表示单个 MCP Server 的当前连接状态
type ServerStatus struct {
	Name      string `json:"name"`       // Server 名称
	Status    string `json:"status"`     // connected / disabled / error
	ToolCount int    `json:"tool_count"` // 已加载的工具数量
	Error     string `json:"error,omitempty"`
}

const (
	StatusConnected    = "connected"
	StatusDisabled     = "disabled"
	StatusError        = "error"
	StatusDisconnected = "disconnected"
)
