// Package questionhandler 提供需求澄清问答回复的 HTTP 客户端方法。
package questionhandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// QuestionHandler 问答回复 HTTP 客户端。
type QuestionHandler struct {
	client *http.Client
	url    string
}

// New 创建问答回复客户端。
func New(client *http.Client, baseURL string) *QuestionHandler {
	return &QuestionHandler{
		client: client,
		url:    baseURL + "/api/ai/answer",
	}
}

// SendAnswer 向服务端提交用户回答。
func (h *QuestionHandler) SendAnswer(id, answer string, isSupplement bool) error {
	body, err := json.Marshal(map[string]interface{}{
		"id":            id,
		"answer":        answer,
		"is_supplement": isSupplement,
	})
	if err != nil {
		return fmt.Errorf("序列化回答请求失败: %w", err)
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送回答请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("回答请求失败 (status=%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
