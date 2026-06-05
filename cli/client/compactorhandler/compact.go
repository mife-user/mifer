package compactorhandler

import (
	"encoding/json"
	"io"
	"net/http"

	"mifer/pkg/errorer"
)

type compactResp struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// Compact 请求服务端执行上下文压缩，返回结果消息
func (h *CompactHandler) Compact() (string, error) {
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

	var cr compactResp
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if cr.Error != "" {
		return "", errorer.NewF(errorer.ErrServerError, cr.Error)
	}

	return cr.Message, nil
}
