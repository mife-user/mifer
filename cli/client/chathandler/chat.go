package chathandler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type chatReq struct {
	Content string `json:"content"`
}

// Send 发送消息并处理SSE流式响应，每收到一个chunk调用onChunk回调
func (h *ChatHandler) Send(ctx context.Context, content string, onChunk func(event, chunk string) error) error {
	body, err := json.Marshal(chatReq{Content: content})
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := h.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回状态码: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
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
				fmt.Fprintf(os.Stderr, "\n错误: %s\n", errMsg)
				return nil
			}
			if err := onChunk(currentEvent, data); err != nil {
				return err
			}
		}

		currentEvent = "response"
	}

	return scanner.Err()
}
