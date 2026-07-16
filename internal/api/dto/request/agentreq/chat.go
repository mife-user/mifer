package agentreq

type ChatReq struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id"` // 指定会话 ID，非空时自动切换记忆
	Mode      string `json:"mode"`       // 对话模式："plan" 为先制定计划等确认再执行
}
