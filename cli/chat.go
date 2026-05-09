package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type chatReq struct {
	Content string `json:"content"`
}

// chat 发送消息并处理SSE流式响应
func (c *Cli) chat(content string) error {
	body, err := json.Marshal(chatReq{Content: content})
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", c.chatURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回状态码: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		switch data {
		case "[DONE]":
			fmt.Println()
			return nil
		case "[ERROR]", "":
			// 忽略空错误
		default:
			if strings.HasPrefix(data, "[ERROR]") {
				errMsg := strings.TrimPrefix(data, "[ERROR] ")
				fmt.Fprintf(os.Stderr, "\n错误: %s\n", errMsg)
				return nil
			}
			fmt.Print(data)
		}
	}

	return scanner.Err()
}
