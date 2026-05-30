package mcphandler

import "net/http"

// MCPHandler MCP状态查询API客户端
type MCPHandler struct {
	http *http.Client
	url  string
}

// New 创建MCPHandler实例
func New(httpClient *http.Client, url string) *MCPHandler {
	return &MCPHandler{http: httpClient, url: url}
}
