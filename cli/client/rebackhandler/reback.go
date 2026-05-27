package rebackhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"mifer/pkg/errorer"
	"net/http"
	"strconv"
)

// RebackEntry 服务器返回的可回退轮次条目
type RebackEntry struct {
	Index   int    `json:"index"`
	Summary string `json:"summary"`
}

type rebackListResp struct {
	Entries []RebackEntry `json:"entries"`
	Error   string        `json:"error,omitempty"`
}

type rebackResp struct {
	Summary string `json:"summary"`
	Content string `json:"content"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// List 获取当前会话中所有可回退的对话轮次
func (h *RebackHandler) List() ([]RebackEntry, error) {
	resp, err := h.http.Get(h.url)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errorer.NewF(errorer.ErrServerStatusCodeDetail, resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	var listResp rebackListResp
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if listResp.Error != "" {
		return nil, errorer.NewF(errorer.ErrServerError, listResp.Error)
	}

	return listResp.Entries, nil
}

// Reback 回退到指定轮次之前，返回系统消息内容和被删除的用户消息完整内容
func (h *RebackHandler) Reback(index int) (string, string, error) {
	req, err := http.NewRequest(http.MethodPost, h.url+"/"+strconv.Itoa(index), nil)
	if err != nil {
		return "", "", errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return "", "", errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", errorer.NewS(errorer.ErrReadResponseFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", errorer.NewF(errorer.ErrServerStatusCodeDetail, resp.StatusCode, string(respBody))
	}

	var rResp rebackResp
	if err := json.Unmarshal(respBody, &rResp); err != nil {
		return "", "", errorer.NewS(errorer.ErrParseResponseFailed, err)
	}

	if rResp.Error != "" {
		return "", "", errorer.NewF(errorer.ErrServerError, rResp.Error)
	}

	return fmt.Sprintf("%s: %s", rResp.Message, rResp.Summary), rResp.Content, nil
}
