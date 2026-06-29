package statushandler

import (
	"encoding/json"
	"io"
	"net/http"

	"mifer/pkg/errorer"
)

// BackendStatus 单个后端的加载状态
type BackendStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "ok" | "failed" | "not_configured"
	Model  string `json:"model,omitempty"`
	Error  string `json:"error,omitempty"`
}

// StatusResp 后端状态查询响应
type StatusResp struct {
	Ready    bool            `json:"ready"`
	Backends []BackendStatus `json:"backends"`
	Warnings []string        `json:"warnings,omitempty"`
}

// StatusHandler 后端状态查询客户端
type StatusHandler struct {
	httpClient *http.Client
	url        string
}

// New 创建 StatusHandler
func New(httpClient *http.Client, baseURL string) *StatusHandler {
	return &StatusHandler{
		httpClient: httpClient,
		url:        baseURL + "/api/admin/status",
	}
}

// Query 查询后端状态
func (h *StatusHandler) Query() (*StatusResp, error) {
	req, err := http.NewRequest("GET", h.url, nil)
	if err != nil {
		return nil, errorer.NewS("创建状态查询请求失败", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errorer.NewS("状态查询请求失败", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorer.NewS("读取状态响应失败", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errorer.NewF("服务器返回状态码: %d", resp.StatusCode)
	}

	var result StatusResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errorer.NewS("解析状态响应失败", err)
	}

	return &result, nil
}
