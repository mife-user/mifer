package confirmhandler

import (
	"bytes"
	"encoding/json"
	"io"
	"mifer/pkg/errorer"
	"net/http"
)

// confirmReq 确认请求
type confirmReq struct {
	CallID string `json:"call_id"`
	Action string `json:"action"`
}

// Confirm 发送工具确认决定到服务端 POST /api/tool/confirm
func (h *ConfirmHandler) Confirm(callID, action string) error {
	body, err := json.Marshal(confirmReq{CallID: callID, Action: action})
	if err != nil {
		return errorer.NewS(errorer.ErrSerializeRequestFailed, err)
	}

	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return errorer.NewF(errorer.ErrServerStatusCode, resp.StatusCode)
	}

	var result struct {
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return errorer.NewS(errorer.ErrParseResponseFailed, err)
	}
	if result.Error != "" {
		return errorer.NewF(errorer.ErrServerError, result.Error)
	}

	return nil
}
