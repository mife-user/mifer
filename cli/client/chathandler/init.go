package chathandler

import "net/http"

// ChatHandler 对话API客户端
type ChatHandler struct {
	http *http.Client
	url  string
}

// New 创建ChatHandler实例
func New(httpClient *http.Client, url string) *ChatHandler {
	return &ChatHandler{http: httpClient, url: url}
}
