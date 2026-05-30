package skillhandler

import "net/http"

// SkillHandler 技能查询API客户端
type SkillHandler struct {
	http *http.Client
	url  string
}

// New 创建SkillHandler实例
func New(httpClient *http.Client, url string) *SkillHandler {
	return &SkillHandler{http: httpClient, url: url}
}
