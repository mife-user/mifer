package conf

import (
	"mifer/pkg/errorer"
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
			return errorer.NewS(errorer.ErrGetHomeDirFailed, err)
		}
		path = filepath.Join(home, "/mifer/config")
		fileName = "prod.yaml"
	}
	// 创建默认配置文件（仅在文件不存在时创建，避免覆盖用户修改）
	if err := os.MkdirAll(path, 0755); err != nil {
		return errorer.NewS(errorer.ErrCreateConfigDirFailed, err)
	}
	cfgPath := filepath.Join(path, fileName)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := os.WriteFile(cfgPath, []byte(defaultConfig), 0644); err != nil {
			return errorer.NewS(errorer.ErrWriteDefaultConfigFailed, err)
		}
	}

	// 创建 .mifer/ 目录及默认 allowlist.yaml（位于工作目录下）
	wd, err := os.Getwd()
	if err != nil {
		return errorer.NewS(errorer.ErrGetWorkDirFailed, err)
	}
	miferDir := filepath.Join(wd, ".mifer")
	if err := os.MkdirAll(miferDir, 0755); err != nil {
		return errorer.NewS(errorer.ErrCreateConfigDirFailed, err)
	}
	allowlistPath := filepath.Join(miferDir, "allowlist.yaml")
	if _, err := os.Stat(allowlistPath); os.IsNotExist(err) {
		if err := os.WriteFile(allowlistPath, []byte(defaultAllowList), 0644); err != nil {
			return errorer.NewS(errorer.ErrWriteDefaultConfigFailed, err)
		}
	}

	return nil
}

const defaultAllowList = `# 命令执行白名单
# 格式：允许的命令列表，为空时不启用白名单检查
# 支持 * 通配符前缀匹配，如 "git*" 匹配所有 git 开头的命令
allow_list:
# 示例：
#   - "ls"
#   - "cat"
#   - "git*"
#   - "python*"
`

const defaultConfig = `
env: dev
log:
  max_size: 100
  max_backups: 7
  level: ""
jwt:
  secret: "123456"
rag:
  chunk_size: 500
  chunk_overlap: 50
  top_k: 5
  dim: 768
  qdrant_host: "localhost"
  qdrant_port: 6334
  qdrant_collection: "mifer_docs"
  qdrant_api_key: ""
mcp:
  servers:
    # 内置 MCP 演示工具：echo / get_time / calculator / random_number
    - name: "demo"
      command: "./test/mcp-demo.exe"
      args: []
      agents: ["MiEditer"]
      enabled: true
ai:
  backends:
    default:
      provider: "openai"
      base_url: "https://api.deepseek.com"
      model: "deepseek-v4-flash"
      api_key: ""
    multi_modal:
      provider: "openai"
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      model: "qwen-omni-turbo"
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
    completable_commands:
      - command: "/help"
        description: "显示帮助信息"
      - command: "/exit"
        description: "退出程序"
      - command: "/quit"
        description: "退出程序"
      - command: "/viewmemory"
        description: "查看/切换对话记忆"
      - command: "/excmem"
        description: "切换到指定记忆"
      - command: "/reback"
        description: "回退对话到历史轮次"
      - command: "/clear"
        description: "创建并切换到新会话"
      - command: "/prompt"
        description: "查看/修改系统提示词"
      - command: "/reload"
        description: "热重载配置与模型"
      - command: "/plan"
        description: "列出/编写项目计划"
      - command: "/mcp"
        description: "显示MCP Server状态"
    content_margin: 4
    min_height: 10
    spinner_type: "MiniDot"
    spinner_frames: []
    spinner_fps: 0
    sidebar_max_log: 100
    sidebar_show_tokens: true
    sidebar_show_timing: true
    completion_max_visible: 5
    mouse_wheel_delta: 3
    horizontal_scroll_step: 4
`
