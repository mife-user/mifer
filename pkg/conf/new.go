package conf

import (
	"fmt"
	"os"
	"path/filepath"
)

func newDefaultCfg(s string) error {
	var path string
	var fileName string
	if s == "dev" {
		path = "./config"
		fileName = "dev.yaml"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败：%w", err)
		}
		path = filepath.Join(home, "/mifer/config")
		fileName = "prod.yaml"
	}
	// 创建默认配置文件（仅在文件不存在时创建，避免覆盖用户修改）
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败：%w", err)
	}
	cfgPath := filepath.Join(path, fileName)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := os.WriteFile(cfgPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("写入默认配置失败：%w", err)
		}
	}
	return nil
}

const defaultConfig = `
env: dev
log:
  max_size: 100
  max_backups: 7
  level: ""
redis:
  host: "127.0.0.1"
  port: "6379"
  username: "mifer"
  password: "mifer"
  db: 0
jwt:
  secret: "123456"
ai:
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  api_key: "your_api_key"
gin:
  mode: "debug"
  port: 15555
  cors:
    allow_origins: ["*"]
    allow_methods: ["POST","GET"]
cli:
  host: "127.0.0.1"
  port: 15555
  lip:
    base:
      foreground: "#00ff11"
      background: "#2c2c2cff"
      bold_top: "#00ff11"
      bold_left: "#00ff11"
      bold_right: "#00ff11"
      bold_bottom: "#00ff11"
    title:
      foreground: "#00D787"
    content:
      foreground: "#FFB86C"
    err:
      foreground: "#FF5555"
    help:
      foreground: "#8BE9FD"
`
