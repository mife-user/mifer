package qq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"mifer/pkg/logger"
)

// OnebotHTTPSender 通过 OneBot HTTP API 发送 QQ 消息，实现 Sender 接口。
// 供 internal/ai/tools/qq 的 qq_send_message 工具在 Agent 调用时发消息。
type OnebotHTTPSender struct {
	httpURL string
	token   string
}

// NewOnebotHTTPSender 创建 OneBot HTTP 消息发送器。
func NewOnebotHTTPSender(httpURL, token string) *OnebotHTTPSender {
	return &OnebotHTTPSender{httpURL: httpURL, token: token}
}

// SendPrivateMsg 发送私聊消息。
func (s *OnebotHTTPSender) SendPrivateMsg(userID int64, message string) error {
	return s.call("send_private_msg", map[string]interface{}{
		"user_id":     userID,
		"message":     message,
		"auto_escape": false,
	})
}

// SendGroupMsg 发送群聊消息。
func (s *OnebotHTTPSender) SendGroupMsg(groupID int64, message string) error {
	return s.call("send_group_msg", map[string]interface{}{
		"group_id":    groupID,
		"message":     message,
		"auto_escape": false,
	})
}

// call 调用 OneBot HTTP API。
func (s *OnebotHTTPSender) call(action string, params map[string]interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{
		"action": action,
		"params": params,
	})
	req, err := http.NewRequest("POST", s.httpURL+"/"+action, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建 OneBot 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("OneBot API 调用失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Warn("OneBot API 返回非200", logger.I("status", resp.StatusCode), logger.S("action", action))
	}
	return nil
}
