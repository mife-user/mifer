package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type memoryResp struct {
	Memory string `json:"memory"`
	Error  string `json:"error,omitempty"`
}

// viewMemory 从API获取并显示会话记忆
func (c *Cli) viewMemory(id string) error {
	url := c.api.memory
	if id != "" {
		url = url + "/" + id
	} else {
		url = url + "/default"
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var memResp memoryResp
	if err := json.Unmarshal(body, &memResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if memResp.Error != "" {
		return fmt.Errorf("服务器错误: %s", memResp.Error)
	}

	if memResp.Memory == "" {
		fmt.Println("(暂无对话记忆)")
	} else {
		fmt.Println("═══════════ 对话记忆 ═══════════")
		fmt.Print(memResp.Memory)
		fmt.Println("═══════════════════════════════")
	}
	return nil
}
