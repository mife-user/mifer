package chathandler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"mifer/pkg/errorer"
	"net/http"
	"strings"
)

type chatReq struct {
	Content string `json:"content"`
	Agent   string `json:"agent,omitempty"`
}

// Send 发送消息并处理SSE流式响应，每收到一个chunk调用onChunk回调。
// agent 为可选的目标 Agent 名称（@AgentName 路由），为空时使用 Orchestrator。
func (h *ChatHandler) Send(ctx context.Context, content string, agent string, onChunk func(event, chunk string) error) error {
	body, err := json.Marshal(chatReq{Content: content, Agent: agent})
	if err != nil {
		return errorer.NewS(errorer.ErrSerializeRequestFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return errorer.NewS(errorer.ErrCreateRequestFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := h.http.Do(req)
	if err != nil {
		return errorer.NewS(errorer.ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errorer.NewF(errorer.ErrServerStatusCode, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// 增加缓冲区上限至 10MB，处理超大 SSE data 行（如 tool_confirm 携带大参数）
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	currentEvent := "response"
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// 仅在 response 事件时反转义服务端转义的换行符
		// 其他事件（如 tool_confirm）的 data 是结构化 JSON，不能做此替换
		if currentEvent == "response" {
			data = strings.ReplaceAll(data, "\\n", "\n")
		}

		switch data {
		case "[DONE]":
			return nil
		case "[ERROR]", "":
			// 忽略空错误
		default:
			if strings.HasPrefix(data, "[ERROR]") {
				errMsg := strings.TrimPrefix(data, "[ERROR] ")
				return errorer.New(errMsg)
			}
			if err := onChunk(currentEvent, data); err != nil {
				return err
			}
		}

		currentEvent = "response"
	}

	return scanner.Err()
}
