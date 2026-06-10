package agentshandler

import "net/http"

// AgentsHandler Agent列表查询API客户端
type AgentsHandler struct {
	http *http.Client
	url  string
}

// New 创建AgentsHandler实例
func New(httpClient *http.Client, url string) *AgentsHandler {
	return &AgentsHandler{http: httpClient, url: url}
}
