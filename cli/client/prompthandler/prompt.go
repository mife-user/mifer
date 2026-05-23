package prompthandler

import (
	"bytes"
	"encoding/json"
	"io"
	"mifer/pkg/errorer"
	"net/http"
)

type promptResp struct {
	Prompt string `json:"prompt"`
	Error  string `json:"error,omitempty"`
}

// Get 获取当前系统提示词
func (h *PromptHandler) Get() (string, error) {
	resp, err := h.http.Get(h.url)
	if err != nil {
		return "", errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var pr promptResp
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if pr.Error != "" {
		return "", errorer.NewF(errorer.ErrServerError, pr.Error)
	}

	return pr.Prompt, nil
}

// Set 设置自定义系统提示词
func (h *PromptHandler) Set(text string) error {
	body, err := json.Marshal(map[string]string{"prompt": text})
	if err != nil {
		return errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}

	resp, err := h.http.Post(h.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var pr promptResp
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if pr.Error != "" {
		return errorer.NewF(errorer.ErrServerError, pr.Error)
	}

	return nil
}

// Reset 重置为默认系统提示词
func (h *PromptHandler) Reset() error {
	req, err := http.NewRequest(http.MethodPost, h.url+"/reset", nil)
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

	var pr promptResp
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if pr.Error != "" {
		return errorer.NewF(errorer.ErrServerError, pr.Error)
	}

	return nil
}
