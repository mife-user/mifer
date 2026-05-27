package confirmhandler

import "net/http"

// ConfirmHandler 工具确认API客户端
type ConfirmHandler struct {
	http *http.Client
	url  string
}

// New 创建ConfirmHandler实例
func New(httpClient *http.Client, url string) *ConfirmHandler {
	return &ConfirmHandler{http: httpClient, url: url}
}
