package excmemhandler

import (
	"encoding/json"
	"fmt"
	"io"
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
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var emResp excmemResp
	if err := json.Unmarshal(respBody, &emResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if emResp.Error != "" {
		return fmt.Errorf("服务器错误: %s", emResp.Error)
	}

	return nil
}
