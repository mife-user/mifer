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
# ── 运行环境 ──
env: dev                             # 运行环境：dev / prod

# ── 日志 ──
log:
  max_size: 100                      # 单文件最大大小（MB）
  max_backups: 7                     # 保留备份文件数量
  level: ""                          # 日志级别，为空时由 env 决定

# ── JWT ──
jwt:
  secret: "123456"                   # JWT 签名密钥

# ── RAG 检索增强 ──
rag:
  chunk_size: 500                    # 文档分块大小（字符数）
  chunk_overlap: 50                  # 相邻分块重叠字符数
  top_k: 5                           # 检索返回结果数量
  dim: 768                           # 向量维度（需匹配嵌入模型）
  qdrant_host: "localhost"           # Qdrant 服务地址
  qdrant_port: 6334                  # Qdrant gRPC 端口
  qdrant_collection: "mifer_docs"    # Qdrant 集合名称
  qdrant_api_key: ""                 # Qdrant API Key（可选）

# ── MCP 服务器 ──
mcp:
  servers:
    - name: "demo"                   # 唯一标识
      command: "./mcp-demo.exe"      # 启动命令
      args: []                       # 命令参数
      agents: ["MiEditer"]           # 分配给哪些 Agent，空或 ["*"] 表示全部
      enabled: true                  # 是否启用

# ── 网页搜索 ──
search:
  provider: ""                       # 搜索方式：searxng / bing / duckduckgo，留空自动
  api_key: ""                        # API 密钥（仅 bing 需要）
  base_url: "http://localhost:18080" # 自定义搜索地址
  max_results: 5                     # 单次最大搜索结果数
  timeout_sec: 10                    # 请求超时（秒）

# ── 技能系统 ──
skill:
  path: ""                           # 技能目录路径，为空时使用默认路径
  enabled: true                      # 是否启用

# ── 工具确认 ──
confirm:
  enabled: true                      # 是否启用工具调用确认
  timeout_sec: 60                    # 确认超时（秒）
  exclude:                           # 无需确认的工具名单
    - knowledge_search
    - file_reader
    - file_viewer

# ── AI 模型后端 ──
ai:
  backends:
    default:                         # 默认模型（fallback）
      provider: "openai"
      base_url: "https://api.deepseek.com"
      model: "deepseek-v4-flash"
      api_key: ""
    multi_modal:                     # 多模态模型（图片识别）
      provider: "openai"
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      model: "qwen-omni-turbo"
      api_key: ""
    haiku:                           # 轻量快速模型
      provider: "openai"
      base_url: "https://api.deepseek.com"
      model: "deepseek-v4-flash"
      api_key: ""
    sonnet:                          # 均衡模型
      provider: "claude"
      model: "claude-sonnet-4-6"
      api_key: ""
    opus:                            # 最强推理模型
      provider: "claude"
      model: "claude-opus-4-7"
      api_key: ""
    embedder:                        # 文本嵌入模型
      provider: "ollama"
      base_url: "http://localhost:11434"
      model: "nomic-embed-text"
      api_key: "ollama"

  # ── 上下文压缩 ──
  context:
    length: 1000000                   # 上下文长度阈值（token数）
    threshold: 0.8                    # 触发压缩的比例
    model: "haiku"                    # 压缩用模型后端
    recent_rounds: 3                  # 压缩后保留的最近轮数

agents:
  - name: "MiTest"                 # Agent名称，唯一标识
    description: "测试Agent" # Agent描述
    instruction: "你是MiTest，测试Agent。" # Agent系统提示词
    model: "sonnet"            # 使用的聊天模型后端，需在 ai.backends 中定义
    tools:                         # 可用工具列表，工具名需在代码中定义
      - file_reader
      
# ── HTTP 服务 ──
gin:
  mode: "debug"                      # 运行模式：debug / release / test
  port: 15555                        # 监听端口
  cors:
    allow_origins: ["*"]             # 允许的跨域来源
    allow_methods: ["POST","GET"]    # 允许的 HTTP 方法

# ── CLI 客户端 ──
cli:
  host: "127.0.0.1"                  # 服务端地址
  port: 15555                        # 服务端端口
  lip:                               # 终端颜色配置
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
    max_history: 100                 # 最大输入历史条数
    completable_commands:            # Tab 补全命令列表
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
      - command: "/config"
        description: "编辑配置文件"
      - command: "/plan"
        description: "列出/编写项目计划"
      - command: "/init"
        description: "生成项目MIFER.md提示词"
      - command: "/mcp"
        description: "显示MCP Server状态"
      - command: "/skill"
        description: "显示已加载的技能列表"
      - command: "/agents"
        description: "显示已配置的Agent列表"
      - command: "/compact"
        description: "手动压缩当前对话上下文"
    content_margin: 4                # 内容区域水平边距
    min_height: 10                   # 终端最小高度（行）
    editor: ""                       # 外部编辑器命令（如 "code --wait"），为空时自动检测
    spinner_type: "MiniDot"          # 加载动画类型
    spinner_frames: []               # 自定义动画帧序列，非空时覆盖 spinner_type
    spinner_fps: 0                   # 自定义动画帧率，0 使用默认值
    sidebar_max_log: 100             # 侧边栏状态日志最大行数
    sidebar_show_tokens: true        # 侧边栏是否显示 token 统计
    sidebar_show_timing: true        # 侧边栏是否显示时间戳
    completion_max_visible: 5        # 补全列表最大可见行数
    mouse_wheel_delta: 3             # 滚轮每次滚动行数
    horizontal_scroll_step: 4        # 水平滚动每次列数
`
