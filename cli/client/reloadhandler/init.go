package reloadhandler

import "net/http"

// ReloadHandler 配置重载API客户端
type ReloadHandler struct {
	http *http.Client
	url  string
}

// New 创建ReloadHandler实例
func New(httpClient *http.Client, url string) *ReloadHandler {
	return &ReloadHandler{http: httpClient, url: url}
}
