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

type memoryListResp struct {
	Current string   `json:"current"`
	IDs     []string `json:"ids"`
	Error   string   `json:"error,omitempty"`
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

// List 获取所有可用记忆ID及当前记忆ID
func (h *MemHandler) List() (current string, ids []string, err error) {
	resp, err := h.http.Get(h.url)
	if err != nil {
		return "", nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("服务器返回状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var listResp memoryListResp
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return "", nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if listResp.Error != "" {
		return "", nil, fmt.Errorf("服务器错误: %s", listResp.Error)
	}

	return listResp.Current, listResp.IDs, nil
}
