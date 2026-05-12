package chathandler

import "net/http"

// ChatHandler 对话API客户端
type ChatHandler struct {
	http    *http.Client
	baseURL string
}

// New 创建ChatHandler实例
func New(httpClient *http.Client, baseURL string) *ChatHandler {
	return &ChatHandler{http: httpClient, baseURL: baseURL}
}
