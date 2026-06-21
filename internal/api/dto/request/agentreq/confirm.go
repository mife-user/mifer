package agentreq

// ConfirmReq 工具确认请求体，POST /api/tool/confirm
type ConfirmReq struct {
	ID     string `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"` // "confirm" | "deny" | "allow"
}
