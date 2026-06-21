package agentreq

// AddAllowListReq 添加命令到白名单的请求体。
type AddAllowListReq struct {
	Command string `json:"command" binding:"required"`
}
