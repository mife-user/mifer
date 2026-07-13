package renamehandler

import "net/http"

// RenameHandler 会话重命名 HTTP 客户端
type RenameHandler struct {
	http *http.Client
	url  string
}

// New 创建 RenameHandler 实例
func New(httpClient *http.Client, url string) *RenameHandler {
	return &RenameHandler{http: httpClient, url: url}
}
