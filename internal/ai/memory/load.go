package memory

import (
	"encoding/json"
	"fmt"
	"mifer/pkg/conf"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

// load 加载记忆数据
func load(config *conf.Config, id []byte) ([]*schema.Message, error) {
	var path string

	if config.Env == "dev" {
		path = filepath.Join("./memory", filepath.Base(config.Workdir))
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败：%w", err)
		}
		path = filepath.Join(home, "/mifer/memory", filepath.Base(config.Workdir))
	}

	// 创建文件夹
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("创建内存目录失败：%w", err)
	}

	// 读取指定下的 JSON 文件
	var messages []*schema.Message
	fileName := filepath.Join(path, fmt.Sprintf("%s.json", id))
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败：%w", err)
	}
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败：%w", err)
	}
	return messages, nil

}
