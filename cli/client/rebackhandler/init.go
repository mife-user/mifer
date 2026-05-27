package rebackhandler

import "net/http"

// RebackHandler 回退对话API客户端
type RebackHandler struct {
	http *http.Client
	url  string
}

// New 创建RebackHandler实例
func New(httpClient *http.Client, url string) *RebackHandler {
	return &RebackHandler{http: httpClient, url: url}
}
