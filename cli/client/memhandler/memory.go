package memhandler

import (
	"encoding/json"
	"io"
	"mifer/pkg/errorer"
	"net/http"
)

type memoryResp struct {
	Memory string `json:"memory"`
	Error  string `json:"error,omitempty"`
}

type memoryListResp struct {
	Current string   `json:"current"`
	IDs     []string `json:"ids"`
	Error   string   `json:"error,omitempty"`
}

// Load 获取指定ID的对话记忆
func (h *MemHandler) Load(id string) (string, error) {
	resp, err := h.http.Get(h.url + "/" + id)
	if err != nil {
		return "", errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errorer.NewF(errorer.ErrServerStatusCodeDetail, resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var memResp memoryResp
	if err := json.Unmarshal(respBody, &memResp); err != nil {
		return "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if memResp.Error != "" {
		return "", errorer.NewF(errorer.ErrServerError, memResp.Error)
	}

	return memResp.Memory, nil
}

// List 获取所有可用记忆ID及当前记忆ID
func (h *MemHandler) List() (current string, ids []string, err error) {
	resp, err := h.http.Get(h.url)
	if err != nil {
		return "", nil, errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, errorer.NewF(errorer.ErrServerStatusCodeDetail, resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var listResp memoryListResp
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return "", nil, errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if listResp.Error != "" {
		return "", nil, errorer.NewF(errorer.ErrServerError, listResp.Error)
	}

	return listResp.Current, listResp.IDs, nil
}
