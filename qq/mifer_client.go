package qq

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mifer/pkg/logger"
)

// miferClient Mifer HTTP API 客户端，封装对话、记忆切换、工具确认的 HTTP 调用。
type miferClient struct {
	baseURL string
}

// ─────────────────────────── 记忆会话切换 ───────────────────────────

// exchangeMemory 切换到指定记忆会话。
// POST {baseURL}/api/memory/exchange/{sessionID}
func (c *miferClient) exchangeMemory(sessionID string) error {
	url := fmt.Sprintf("%s/api/memory/exchange/%s", c.baseURL, sessionID)
	resp, err := httpClient.Post(url, "application/json", nil)
	if err != nil {
		logger.Error("QQ切换记忆会话失败", logger.S("session", sessionID), logger.C(err))
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Warn("QQ切换记忆会话返回非200", logger.I("status", resp.StatusCode), logger.S("session", sessionID))
	}
	return nil
}

// ─────────────────────────── SSE 对话 ───────────────────────────

// sseCallback SSE 事件回调，eventType 为 response/tool_confirm/agent_start 等。
type sseCallback func(eventType, data string) error

// chat 发送对话请求并读取 SSE 流。
// POST {baseURL}/api/ai/chat  body: {"content": query}
func (c *miferClient) chat(query string, cb sseCallback) error {
	body, _ := json.Marshal(map[string]string{"content": query})
	url := c.baseURL + "/api/ai/chat"

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Error("QQ发送对话请求失败", logger.C(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		logger.Error("QQ对话请求返回错误", logger.I("status", resp.StatusCode), logger.S("body", string(bodyBytes)))
		return fmt.Errorf("chat 请求失败，状态码: %d", resp.StatusCode)
	}

	return c.readSSE(resp.Body, cb)
}

// readSSE 解析 SSE 流，逐事件调用回调。
func (c *miferClient) readSSE(r io.Reader, cb sseCallback) error {
	scanner := bufio.NewScanner(r)
	var eventType, data string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// 空行表示一个事件结束
			if eventType != "" && data != "" {
				if data == "[DONE]" {
					return nil
				}
				if strings.HasPrefix(data, "[ERROR]") {
					logger.Warn("QQ对话SSE错误", logger.S("error", data))
					return nil
				}
				if err := cb(eventType, data); err != nil {
					return err
				}
			}
			eventType = ""
			data = ""
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
	}

	return scanner.Err()
}

// ─────────────────────────── 工具确认 ───────────────────────────

// allowedTools QQ 通道允许自动通过的工具名集合。
var allowedTools = map[string]bool{
	"qq_send_message": true,
}

// confirmTool 自动处理工具确认。
// 若工具在 allowedTools 中则确认，否则拒绝。
func (c *miferClient) confirmTool(eventData string) {
	// 解析 tool_confirm 事件数据：格式为 "tool_name\x00{json_args}"
	parts := strings.SplitN(eventData, "\x00", 2)
	toolName := ""
	if len(parts) > 0 {
		toolName = parts[0]
	}

	action := "deny"
	if allowedTools[toolName] {
		action = "confirm"
	}

	// 从 eventData 中提取 confirm_id（位于 JSON 部分的 id 字段）
	confirmID := extractConfirmID(eventData)
	if confirmID == "" {
		logger.Warn("QQ无法解析工具确认ID", logger.S("tool", toolName))
		return
	}

	body, _ := json.Marshal(map[string]string{
		"id":     confirmID,
		"action": action,
	})
	url := c.baseURL + "/api/tool/confirm"
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Error("QQ发送工具确认失败", logger.S("tool", toolName), logger.C(err))
		return
	}
	resp.Body.Close()

	logger.Debug("QQ工具确认", logger.S("tool", toolName), logger.S("action", action))
}

// extractConfirmID 从 SSE event data 中提取 confirm_id。
// tool_confirm 事件 data 格式为 "tool_name\x00{json}"，其中 json 包含 "id" 字段。
func extractConfirmID(eventData string) string {
	parts := strings.SplitN(eventData, "\x00", 2)
	if len(parts) < 2 {
		return ""
	}
	var confirm struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(parts[1]), &confirm); err != nil {
		return ""
	}
	return confirm.ID
}
