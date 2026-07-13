package renamehandler

import (
	"bytes"
	"encoding/json"
	"io"
	"mifer/pkg/errorer"
	"net/http"
)

type renameReq struct {
	Name string `json:"name"`
}

type renameResp struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// Rename 请求服务端重命名当前会话
func (h *RenameHandler) Rename(name string) (string, error) {
	body, err := json.Marshal(renameReq{Name: name})
	if err != nil {
		return "", errorer.NewS(errorer.ErrSerializeRequestFailed, err)
	}

	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(body))
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

	var rr renameResp
	if err := json.Unmarshal(respBody, &rr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}
	if rr.Error != "" {
		return "", errorer.NewF(errorer.ErrServerError, rr.Error)
	}

	return rr.Message, nil
}
