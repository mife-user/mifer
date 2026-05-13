package excmemhandler

import "net/http"

// ExcmemHandler 记忆交换API客户端
type ExcmemHandler struct {
	http *http.Client
	url  string
}

// New 创建ExcmemHandler实例
func New(httpClient *http.Client, url string) *ExcmemHandler {
	return &ExcmemHandler{http: httpClient, url: url}
}
