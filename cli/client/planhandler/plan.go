package planhandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// PlanHandler 计划文件 HTTP 客户端
type PlanHandler struct {
	http *http.Client
	url  string
}

// planListResp 服务端计划列表响应
type planListResp struct {
	Files []string `json:"files"`
	Error string   `json:"error,omitempty"`
}

// planLoadResp 服务端计划内容响应
type planLoadResp struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// New 创建 PlanHandler 实例
func New(httpClient *http.Client, url string) *PlanHandler {
	return &PlanHandler{http: httpClient, url: url}
}

// List 列出所有计划文件
func (h *PlanHandler) List() ([]string, error) {
	resp, err := h.http.Get(h.url)
	if err != nil {
		return nil, fmt.Errorf("请求计划列表失败: %w", err)
	}
	defer resp.Body.Close()

	var result planListResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析计划列表响应失败: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("服务端错误: %s", result.Error)
	}
	return result.Files, nil
}

// Load 加载指定计划文件内容
func (h *PlanHandler) Load(name string) (string, error) {
	requestURL := h.url + "/" + url.PathEscape(name)
	resp, err := h.http.Get(requestURL)
	if err != nil {
		return "", fmt.Errorf("请求计划文件失败: %w", err)
	}
	defer resp.Body.Close()

	var result planLoadResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析计划文件响应失败: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("服务端错误: %s", result.Error)
	}
	return result.Content, nil
}
