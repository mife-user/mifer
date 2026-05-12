package memhandler

import "net/http"

// MemHandler 记忆API客户端
type MemHandler struct {
	http    *http.Client
	baseURL string
}

// New 创建MemHandler实例
func New(httpClient *http.Client, baseURL string) *MemHandler {
	return &MemHandler{http: httpClient, baseURL: baseURL}
}
