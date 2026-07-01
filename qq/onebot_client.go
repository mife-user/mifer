package qq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mifer/pkg/logger"
)

// onebotClient OneBot HTTP API 客户端，用于调用 SnowLuma 的 HTTP API 发送 QQ 消息。
type onebotClient struct {
	httpURL string
	token   string
}

// ─────────────────────────── 消息发送 ───────────────────────────

// sendPrivateMsg 发送私聊消息。
func (c *onebotClient) sendPrivateMsg(userID int64, message string) error {
	return c.callAPI("send_private_msg", map[string]interface{}{
		"user_id":     userID,
		"message":     message,
		"auto_escape": false,
	})
}

// sendGroupMsg 发送群聊消息。
func (c *onebotClient) sendGroupMsg(groupID int64, message string) error {
	return c.callAPI("send_group_msg", map[string]interface{}{
		"group_id":    groupID,
		"message":     message,
		"auto_escape": false,
	})
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

// ─────────────────────────── OneBot API 通用调用 ───────────────────────────

// callAPI 通用 OneBot HTTP API 调用。
// POST {httpURL}/{action}
func (c *onebotClient) callAPI(action string, params map[string]interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"action": action,
		"params": params,
	})
	if err != nil {
		return fmt.Errorf("序列化 OneBot 请求失败: %w", err)
	}

	url := c.httpURL + "/" + action
	logger.Info("QQ OneBot HTTP 请求",
		logger.S("action", action),
		logger.S("url", url),
		logger.S("body", string(body)),
	)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建 OneBot 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("QQ OneBot HTTP 调用失败", logger.S("url", url), logger.C(err))
		return fmt.Errorf("OneBot API 调用失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	logger.Info("QQ OneBot HTTP 响应",
		logger.I("status", resp.StatusCode),
		logger.S("body", string(respBody)),
	)

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		logger.Warn("QQ OneBot 响应解析失败", logger.C(err))
		return fmt.Errorf("解析 OneBot 响应失败: %w", err)
	}
	if apiResp.Status != "ok" {
		logger.Warn("QQ OneBot 返回非 ok",
			logger.S("status", apiResp.Status),
			logger.I("retcode", apiResp.RetCode),
		)
		return fmt.Errorf("OneBot 返回错误: retcode=%d, status=%s", apiResp.RetCode, apiResp.Status)
	}
	return nil
}

// apiResponse OneBot API 通用响应。
type apiResponse struct {
	Status  string `json:"status"`
	RetCode int    `json:"retcode"`
}
