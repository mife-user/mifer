package agentshandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mifer/pkg/errorer"
)

type agentInfo struct {
	Name         string   `json:"name"`
	ModelBackend string   `json:"model_backend"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Description  string   `json:"description"`
	Tools        []string `json:"tools"`
}

type listResp struct {
	Agents []agentInfo `json:"agents"`
}

// List 查询Agent列表，返回格式化后的状态信息
func (h *AgentsHandler) List() (string, error) {
	req, err := http.NewRequest(http.MethodGet, h.url, nil)
	if err != nil {
		return "", errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}

	resp, err := h.http.Do(req)
	if err != nil {
		return "", errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var lr listResp
	if err := json.Unmarshal(respBody, &lr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if len(lr.Agents) == 0 {
		return "没有已配置的 Agent", nil
	}

	var sb strings.Builder
	sb.WriteString("已配置的 Agent:")
	for _, a := range lr.Agents {
		modelInfo := fmt.Sprintf("%s/%s", a.Provider, a.Model)
		tc := len(a.Tools)
		if tc > 0 {
			fmt.Fprintf(&sb, "\n  ● %-16s 模型: %-24s  工具: %s", a.Name, modelInfo, strings.Join(a.Tools, ", "))
		} else {
			fmt.Fprintf(&sb, "\n  ● %-16s 模型: %-24s  工具: 无", a.Name, modelInfo)
		}
		if a.Description != "" {
			fmt.Fprintf(&sb, "\n    %s", a.Description)
		}
	}
	return sb.String(), nil
}
