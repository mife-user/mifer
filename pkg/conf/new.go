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
  username: ""
  password: ""
  db: 0
  protocol: 2
  unstable_resp3: true
jwt:
  secret: "123456"
rag:
  chunk_size: 500
  chunk_overlap: 50
  index_name: "mifer_docs"
  key_prefix: "mifer:docs:"
  top_k: 5
  dim: 768
ai:
  backends:
    default:
      provider: "openai"
      base_url: "https://api.deepseek.com"
      model: "deepseek-v4-flash"
      api_key: ""
    multi_modal:
      provider: "gemini"
      model: "gemini-2.5-flash"
      api_key: ""
    haiku:
      provider: "openai"
      base_url: "https://api.deepseek.com"
      model: "deepseek-v4-flash"
      api_key: ""
    sonnet:
      provider: "claude"
      model: "claude-sonnet-4-6"
      api_key: ""
    opus:
      provider: "claude"
      model: "claude-opus-4-7"
      api_key: ""
    embedder:
      provider: "ollama"
      base_url: "http://localhost:11434"
      model: "nomic-embed-text"
      api_key: "ollama"
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
    title:
      foreground: "#00D787"
    content:
      foreground: "#FFB86C"
    err:
      foreground: "#FF5555"
    help:
      foreground: "#8BE9FD"
    think:
      foreground: "#FFB86C"
    scroll:
      foreground: "#666666"
    separator:
      foreground: "#444444"
    sidebar:
      foreground: "#555555"
    sidebar_active:
      foreground: "#00D787"
    sidebar_completed:
      foreground: "#666666"
    sidebar_separator:
      foreground: "#444444"
    sidebar_placeholder:
      foreground: "#555555"
  tui:
    max_history: 100
    completable_commands: ["/help", "/exit", "/quit", "/viewmemory", "/excmem"]
    content_margin: 4
    min_height: 10
    spinner_type: "MiniDot"
    spinner_frames: []
    spinner_fps: 0
    sidebar_max_log: 100
    sidebar_show_tokens: true
    sidebar_show_timing: true
    completion_max_visible: 5
`
