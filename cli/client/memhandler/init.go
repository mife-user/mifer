package memhandler

import "net/http"

// MemHandler 记忆API客户端
type MemHandler struct {
	http *http.Client
	url  string
}

// New 创建MemHandler实例
func New(httpClient *http.Client, url string) *MemHandler {
	return &MemHandler{http: httpClient, url: url}
}
