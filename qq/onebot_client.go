package qq

import (
	"mifer/pkg/logger"
)

// onebotClient OneBot 消息发送器，通过 WebSocket 连接发送 action 而非 HTTP。
// OneBot v11 规范：WebSocket 连接上可双向通信，客户端发送 action JSON 即可调用 API。
type onebotClient struct {
	ws *wsClient // 共用 WebSocket 连接
}

// sendReply 根据事件类型发送回复。
func (c *onebotClient) sendReply(event *oneBotEvent, content string) {
	logger.Info("QQ 发送回复",
		logger.S("type", event.MessageType),
		logger.I("target", int(event.UserID)),
		logger.I("len", len(content)),
	)
	var err error
	switch event.MessageType {
	case "private":
		err = c.sendPrivateMsg(event.UserID, content)
	case "group":
		err = c.sendGroupMsg(event.GroupID, content)
	default:
		logger.Warn("QQ未知消息类型，无法发送回复", logger.S("type", event.MessageType))
		return
	}
	if err != nil {
		logger.Error("QQ发送消息失败", logger.C(err))
	} else {
		logger.Info("QQ 消息发送成功")
	}
}

// sendPrivateMsg 发送私聊消息（通过 WebSocket）。
func (c *onebotClient) sendPrivateMsg(userID int64, message string) error {
	return c.sendAction("send_private_msg", map[string]interface{}{
		"user_id":     userID,
		"message":     message,
		"auto_escape": false,
	})
}

// sendGroupMsg 发送群聊消息（通过 WebSocket）。
func (c *onebotClient) sendGroupMsg(groupID int64, message string) error {
	return c.sendAction("send_group_msg", map[string]interface{}{
		"group_id":    groupID,
		"message":     message,
		"auto_escape": false,
	})
}

// sendAction 通过 WebSocket 发送 OneBot action。
func (c *onebotClient) sendAction(action string, params map[string]interface{}) error {
	logger.Info("QQ OneBot WS 发送", logger.S("action", action))
	msg := map[string]interface{}{
		"action": action,
		"params": params,
	}
	return c.ws.WriteJSON(msg)
}
