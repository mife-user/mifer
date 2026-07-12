package agentreq

type ChatReq struct {
	Content   string `json:"content"`
	Channel   string `json:"channel"`    // 消息来源通道，"qq" 为 QQ 机器人通道
	SessionID string `json:"session_id"` // 指定会话 ID，非空时自动切换记忆
	Mode      string `json:"mode"`       // 对话模式："plan" 为先制定计划等确认再执行
}
