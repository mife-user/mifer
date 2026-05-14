package memhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type memoryResp struct {
	Memory string `json:"memory"`
	Error  string `json:"error,omitempty"`
}

// Load 获取指定ID的对话记忆
func (h *MemHandler) Load(id string) (string, error) {
	resp, err := h.http.Get(h.url + "/" + id)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("服务器返回状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var memResp memoryResp
	if err := json.Unmarshal(respBody, &memResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if memResp.Error != "" {
		return "", fmt.Errorf("服务器错误: %s", memResp.Error)
	}

	return memResp.Memory, nil
}
