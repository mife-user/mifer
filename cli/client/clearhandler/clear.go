package clearhandler

import (
	"encoding/json"
	"io"
	"mifer/pkg/errorer"
	"net/http"
)

type clearResp struct {
	NewID string `json:"new_id"`
	Error string `json:"error,omitempty"`
}

// Clear 请求服务端生成新会话ID并切换，返回新ID
func (h *ClearHandler) Clear() (string, error) {
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

	var cr clearResp
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if cr.Error != "" {
		return "", errorer.NewF(errorer.ErrServerError, cr.Error)
	}

	return cr.NewID, nil
}
