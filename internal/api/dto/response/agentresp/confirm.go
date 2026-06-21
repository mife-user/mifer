package agentres

// ConfirmRes 工具确认响应体
type ConfirmRes struct {
	ID       string `json:"id"`
	Resolved bool   `json:"resolved"`
	Action   string `json:"action"`
}

// AddAllowListRes 添加白名单响应体
type AddAllowListRes struct {
	Command string `json:"command"`
	Added   bool   `json:"added"`
}
