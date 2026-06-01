// Package toolconfirmhandler 提供工具确认相关的 HTTP 客户端方法。
package toolconfirmhandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ConfirmHandler 工具确认 HTTP 客户端。
type ConfirmHandler struct {
	client *http.Client
	url    string
}

// New 创建工具确认客户端。
func New(client *http.Client, baseURL string) *ConfirmHandler {
	return &ConfirmHandler{
		client: client,
		url:    baseURL + "/api/tool/confirm",
	}
}

// Confirm 确认执行指定工具调用。action 为 "confirm"/"deny"/"allow"。
func (h *ConfirmHandler) Confirm(id, action string) error {
	body, err := json.Marshal(map[string]string{
		"id":     id,
		"action": action,
	})
	if err != nil {
		return fmt.Errorf("序列化确认请求失败: %w", err)
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送确认请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("确认请求失败 (status=%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// AllowlistHandler 命令白名单 HTTP 客户端。
type AllowlistHandler struct {
	client *http.Client
	url    string
}

// NewAllowlist 创建白名单客户端。
func NewAllowlist(client *http.Client, baseURL string) *AllowlistHandler {
	return &AllowlistHandler{
		client: client,
		url:    baseURL + "/api/tool/allowlist/add",
	}
}

// Add 添加命令到白名单。
func (h *AllowlistHandler) Add(command string) error {
	body, err := json.Marshal(map[string]string{
		"command": command,
	})
	if err != nil {
		return fmt.Errorf("序列化白名单请求失败: %w", err)
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送白名单请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("白名单请求失败 (status=%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
