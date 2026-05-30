package mcphandler

import (
	"encoding/json"
	"fmt"
	"io"
	"mifer/pkg/errorer"
	"net/http"
	"strings"
)

type serverStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	ToolCount int    `json:"tool_count"`
	Error     string `json:"error,omitempty"`
}

type statusResp struct {
	Servers []serverStatus `json:"servers"`
}

// Status 查询 MCP Server 状态，返回格式化后的状态信息
func (h *MCPHandler) Status() (string, error) {
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

	var sr statusResp
	if err := json.Unmarshal(respBody, &sr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if len(sr.Servers) == 0 {
		return "没有配置 MCP Server", nil
	}

	var sb strings.Builder
	sb.WriteString("MCP Server 状态:")
	for _, s := range sr.Servers {
		icon := "○"
		switch s.Status {
		case "connected":
			icon = "●"
		case "error":
			icon = "✕"
		}
		fmt.Fprintf(&sb, "\n  %s %-16s %-12s %d tools", icon, s.Name, s.Status, s.ToolCount)
		if s.Error != "" {
			sb.WriteString(" — " + s.Error)
		}
	}
	return sb.String(), nil
}
