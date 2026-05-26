package reloadhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"mifer/pkg/errorer"
	"net/http"
	"strings"
)

type backendStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Model  string `json:"model,omitempty"`
	Error  string `json:"error,omitempty"`
}

type reloadResp struct {
	Success  bool            `json:"success"`
	Message  string          `json:"message"`
	Backends []backendStatus `json:"backends"`
	Error    string          `json:"error,omitempty"`
}

// Reload 请求服务端重载配置，返回格式化后的状态信息
func (h *ReloadHandler) Reload() (string, error) {
	req, err := http.NewRequest(http.MethodPost, h.url, nil)
	if err != nil {
		return "", errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return "", errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var rr reloadResp
	if err := json.Unmarshal(respBody, &rr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if rr.Error != "" {
		return "", errorer.NewF(errorer.ErrServerError, rr.Error)
	}

	var sb strings.Builder
	sb.WriteString(rr.Message)
	for _, b := range rr.Backends {
		fmt.Fprintf(&sb, "\n  %-12s %s", b.Name, b.Status)
		if b.Status == "failed" && b.Error != "" {
			sb.WriteString(" — " + b.Error)
		}
	}
	return sb.String(), nil
}
