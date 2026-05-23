package clearhandler

import "net/http"

// ClearHandler 清空记忆API客户端
type ClearHandler struct {
	http *http.Client
	url  string
}

// New 创建ClearHandler实例
func New(httpClient *http.Client, url string) *ClearHandler {
	return &ClearHandler{http: httpClient, url: url}
}
