package compactorhandler

import "net/http"

// CompactHandler 上下文压缩 API 客户端
type CompactHandler struct {
	http *http.Client
	url  string
}

// New 创建 CompactHandler 实例
func New(httpClient *http.Client, url string) *CompactHandler {
	return &CompactHandler{http: httpClient, url: url}
}
