package agentreq

type ChatReq struct {
	Content string `json:"content"`
	Channel string `json:"channel"` // 消息来源通道，"qq" 为 QQ 机器人通道
}
