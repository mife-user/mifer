package excmemhandler

import (
	"encoding/json"
	"io"
	"mifer/pkg/errorer"
	"net/http"
)

type excmemResp struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// Exchange 切换到指定ID的记忆
func (h *ExcmemHandler) Exchange(id string) error {
	req, err := http.NewRequest(http.MethodPost, h.url+"/"+id, nil)
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

	var emResp excmemResp
	if err := json.Unmarshal(respBody, &emResp); err != nil {
		return errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if emResp.Error != "" {
		return errorer.NewF(errorer.ErrServerError, emResp.Error)
	}

	return nil
}
