package prompthandler

import "net/http"

// PromptHandler 提示词API客户端
type PromptHandler struct {
	http *http.Client
	url  string
}

// New 创建PromptHandler实例
func New(httpClient *http.Client, url string) *PromptHandler {
	return &PromptHandler{http: httpClient, url: url}
}
