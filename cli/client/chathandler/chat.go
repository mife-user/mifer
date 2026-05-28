package chathandler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"net/http"
	"strings"
)

type chatReq struct {
	Content  string `json:"content"`
	PlanMode bool   `json:"plan_mode,omitempty"`
}

// Send 发送消息并处理SSE流式响应，每收到一个chunk调用onChunk回调
func (h *ChatHandler) Send(ctx context.Context, content string, onChunk func(event, chunk string) error) error {
	body, err := json.Marshal(chatReq{Content: content})
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
	bufSize := conf.GetConfig().Cli.Tui.SSEScannerBufferSize
	if bufSize <= 0 {
		bufSize = 1048576
	}
	scanner.Buffer(make([]byte, 0, bufSize), bufSize)
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

		// 反转义服务端转义的换行符
		data = strings.ReplaceAll(data, "\\n", "\n")

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

	if err := scanner.Err(); err != nil {
		return errorer.NewS(errorer.ErrSSEScannerFailed, err)
	}
	return nil
}

// SendPlan 以计划模式发送消息，强制服务端走 plan graph
func (h *ChatHandler) SendPlan(ctx context.Context, content string, onChunk func(event, chunk string) error) error {
	body, err := json.Marshal(chatReq{Content: content, PlanMode: true})
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
	bufSize := conf.GetConfig().Cli.Tui.SSEScannerBufferSize
	if bufSize <= 0 {
		bufSize = 1048576
	}
	scanner.Buffer(make([]byte, 0, bufSize), bufSize)
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
		data = strings.ReplaceAll(data, "\\n", "\n")

		switch data {
		case "[DONE]":
			return nil
		case "[ERROR]", "":
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

	if err := scanner.Err(); err != nil {
		return errorer.NewS(errorer.ErrSSEScannerFailed, err)
	}
	return nil
}
