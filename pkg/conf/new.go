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
		path = filepath.Join(home, "/.mifer/config")
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
# ┌──────────────────────────────────────────────────────────────────────┐
# │                        Mifer 配置文件说明                            │
# ├──────────────────────────────────────────────────────────────────────┤
# │                                                                      │
# │  快速入门：                                                          │
# │    1. 在 ai.backends.default.api_key 填入你的 API Key               │
# │    2. 根据需要修改 default.provider 和 default.base_url              │
# │    3. 保存后运行 /reload 或重启程序即可生效                          │
# │                                                                      │
# │  环境变量覆盖（优先级高于本文件）：                                  │
# │    MIFER_AI_DEFAULT_APIKEY    — 覆盖 default 后端 api_key           │
# │    MIFER_AI_DEFAULT_BASE_URL  — 覆盖 default 后端 base_url          │
# │    MIFER_AI_DEFAULT_PROVIDER  — 覆盖 default 后端 provider          │
# │    MIFER_AI_DEFAULT_MODEL     — 覆盖 default 后端 model             │
# │    （同样支持 HAIKU、SONNET、OPUS、MULTI 后缀）                      │
# │    MIFER_ENV=dev 强制使用开发环境配置                                │
# │    MIFER_JWT_SECRET           — 覆盖 JWT 签名密钥                   │
# │    MIFER_SEARCH_API_KEY       — 覆盖搜索引擎 API Key                │
# └──────────────────────────────────────────────────────────────────────┘

# ── 运行环境 ──
# dev  = 开发模式（Debug 日志、./config/dev.yaml、./memory/ 目录）
# prod = 生产模式（Info 日志、~/.mifer/config/prod.yaml、~/.mifer/memory/ 目录）
env: prod

# ── 日志 ──
log:
  max_size: 100                      # 单个日志文件最大大小（MB），超过后自动轮换
  max_backups: 7                     # 最多保留的旧日志文件数量，超出后删除最旧的
  level: ""                          # 日志级别：debug / info / warn / error，为空时 dev=debug, prod=info

# ── JWT 认证 ──
# secret 用于签发和验证 API 访问令牌，生产环境请修改为复杂随机字符串
jwt:
  secret: "123456"

# ── 路径与快照 ──
path:
  snapshot_enabled: true             # 文件快照：每轮对话后自动保存文件状态，回退时可恢复

# ── RAG 检索增强生成 ──
# 依赖 Qdrant 向量数据库 + Ollama 嵌入模型，用于知识库文档的存储和语义检索
# 启动前需运行：docker-compose up -d qdrant ollama
rag:
  chunk_size: 500                    # 文档分块大小（字符数），越大上下文越完整但检索精度越低
  chunk_overlap: 50                  # 相邻分块的重叠字符数，防止关键信息在分块边界被切断
  top_k: 5                           # 知识库检索时返回的最相关文档片段数量
  dim: 768                           # 向量维度，必须与嵌入模型输出维度一致（nomic-embed-text 为 768）
  qdrant_host: "localhost"           # Qdrant 向量数据库地址
  qdrant_port: 6334                  # Qdrant gRPC 端口（HTTP 端口为 6333）
  qdrant_collection: "mifer_docs"    # Qdrant 集合名称，同一实例可共用
  qdrant_api_key: ""                 # Qdrant API Key（Qdrant Cloud 需要，本地部署留空）

# ── MCP 服务器（Model Context Protocol） ──
# MCP 是外部工具扩展协议，可接入第三方工具服务（如 GitHub、数据库、文件系统等）
# 每个 Server 可分配给的 Agent 列表：MiEditer / MiSummarizer / MiPlanner / MiCommander / MiAuditor / Mifer
mcp:
  servers:
    - name: "demo"                   # Server 唯一标识，用于日志和状态查询
      command: "./mcp-demo.exe"      # 启动命令（支持相对路径和绝对路径）
      args: []                       # 命令行参数列表，如 ["--port", "8080"]
      env: []                        # 环境变量，格式：["KEY=VALUE", "TOKEN=xxx"]
      agents: ["MiEditer"]           # 分配给哪些 Agent，["*"] 或不填表示全部可用
      enabled: true                  # true=随主程序自动启动，false=需手动启用

# ── 网页搜索 ──
# 为 AI 提供联网搜索能力，支持三种后端：
#   searxng  — 自托管元搜索引擎（默认，需 docker-compose up -d searxng）
#   bing     — 微软 Bing Search API（需 api_key）
#   duckduckgo — 免费，无需 API Key，但可能被限速
#   留空自动按 searxng > duckduckgo 优先级选择
search:
  provider: ""                       # searxng / bing / duckduckgo / 留空自动
  api_key: ""                        # API 密钥（仅 bing 需要，在 Azure Portal 获取）
  base_url: "http://localhost:18080" # SearXNG 服务地址（docker-compose 默认端口 18080）
  max_results: 5                     # 单次搜索返回的最大结果数
  timeout_sec: 10                    # 单次搜索请求超时秒数

# ── 技能系统 ──
# 技能是预定义的任务模板，存放在 .mifer/skills/ 目录下，每个技能一个 SKILL.md 文件
# AI 可根据用户请求自动匹配并执行相应技能
skill:
  path: ""                           # 技能目录路径，为空时默认读取 <工作目录>/.mifer/skills/
  enabled: true                      # true=启用技能系统，false=禁用

# ── 工具调用确认 ──
# AI 在执行文件写入、命令执行等敏感操作前，需要用户确认，防止误操作
confirm:
  enabled: true                      # true=启用工具确认，false=所有工具自动执行（生产环境建议 true）
  timeout_sec: 60                    # 等待用户确认的超时秒数，超时自动拒绝
  exclude:                           # 无需确认的安全工具（仅读取操作），不会弹出确认框
    - knowledge_search               # 知识库检索
    - file_reader                    # 文件读取
    - file_viewer                    # 文件查看

# ── QQ Bot ──
# QQ 机器人功能，通过 SnowLuma（OneBot v11 协议桥）收发 QQ 消息
qq:
  enabled: false                      # true=启用 QQ Bot，需先启动 SnowLuma 并登录 QQ
  onebot:
    ws_url: "ws://127.0.0.1:3001"    # SnowLuma WebSocket 地址
    http_url: "http://127.0.0.1:3001" # SnowLuma HTTP API 地址
    access_token: ""                  # OneBot access_token（可选，需与 SnowLuma 配置一致）
  bot:
    qq: 0                             # Bot 自己的 QQ 号（必须配置）
    group_reply_mode: "mention_only"  # 群聊回复模式：mention_only（仅被@时回复）/ always（总是回复）
    private_enabled: true             # 是否响应私聊

# ── AI 模型后端 ──
# 每个后端定义一个 AI 模型连接，provider 支持四种类型：
#   openai  — OpenAI 兼容接口（支持 DeepSeek、通义千问、硅基流动等所有 OpenAI-API 兼容服务）
#   claude  — Anthropic Claude 原生接口
#   gemini  — Google Gemini 原生接口
#   ollama  — 本地 Ollama 服务（免费，无需 API Key，适合隐私敏感场景）
#
# 各后端用途说明：
#   default     — 主对话模型，所有请求的默认选择，必须配置
#   multi_modal — 图片识别模型，用于 file_viewer / image_generator 工具
#   haiku       — 轻量模型，用于上下文压缩等低算力任务
#   sonnet      — 均衡模型，用于 MiEditer / MiSummarizer / MiCommander 子 Agent
#   opus        — 最强推理，用于 MiPlanner / MiAuditor 需要深度分析的子 Agent
#   embedder    — 文本嵌入模型，用于 RAG 知识库文档向量化（通常用本地 Ollama）
#
# api_key 获取方式（常见平台）：
#   DeepSeek   → https://platform.deepseek.com/api_keys
#   ￦阿里百炼  → https://bailian.console.aliyun.com/  （通义千问系列）
#   Claude     → https://console.anthropic.com/  （Anthropic API）
#   OpenAI     → https://platform.openai.com/api-keys
#   Gemini     → https://aistudio.google.com/apikey
#   Ollama     → 本地运行，api_key 可任意填写（如 "ollama"）
ai:
  backends:
    default:                         # 【必填】主调度模型，Mifer 编排器使用
      provider: "openai"             # openai / claude / gemini / ollama
      base_url: "https://api.deepseek.com"  # API 地址，支持任意 OpenAI 兼容端点
      model: "deepseek-v4-flash"     # 模型名称，需与你申请的 API 平台一致
      api_key: ""                    # 【必填】API Key，在此填入或设置环境变量 MIFER_AI_DEFAULT_APIKEY
    multi_modal:                     # 多模态模型，用于图片识别和图片生成
      provider: "openai"
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      model: "qwen-omni-turbo"       # 通义千问多模态模型，支持图片理解
      api_key: ""                    # 阿里云百炼 API Key，留空则图片功能不可用
    haiku:                           # 轻量快速模型，用于上下文压缩等低算力场景
      provider: "openai"
      base_url: "https://api.deepseek.com"
      model: "deepseek-v4-flash"
      api_key: ""                    # 为空时自动回退到 default 后端
    sonnet:                          # 均衡模型，用于文件编辑、摘要、命令执行等子 Agent
      provider: "claude"
      model: "claude-sonnet-4-6"
      api_key: ""                    # 为空时自动回退到 default 后端
    opus:                            # 最强推理模型，用于计划编写和安全审计
      provider: "claude"
      model: "claude-opus-4-7"
      api_key: ""                    # 为空时自动回退到 default 后端
    embedder:                        # 文本嵌入模型，将文档转为向量存入 Qdrant
      provider: "ollama"             # 推荐使用本地 Ollama（免费，数据不出本机）
      base_url: "http://localhost:11434"  # Ollama 默认地址
      model: "nomic-embed-text"      # 轻量嵌入模型（需先运行 ollama pull nomic-embed-text）
      api_key: "ollama"              # Ollama 无需真实 API Key，任意字符串即可

  # ── 上下文压缩 ──
  # 当对话历史超过阈值时，自动使用 haiku 模型将早期对话压缩为摘要
  # 压缩后仍保留最近 recent_rounds 轮完整对话
  context:
    length: 1000000                  # 上下文长度阈值（Prompt Tokens 数）
    threshold: 0.8                   # 触发压缩的比例（当前 Tokens > length * threshold 时压缩）
    model: "haiku"                   # 执行压缩的模型后端（建议用 haiku 减少耗时）
    recent_rounds: 3                 # 压缩后保留的最近对话轮数

# ── 自定义 Agent ──
# 除内置的 5 个子 Agent 外，可在下方定义额外的专用 Agent
# model 字段必须是 ai.backends 中已定义的后端名称
# tools 支持：file_reader, file_writer, file_creator, file_viewer, image_generator,
#            knowledge_search, knowledge_store, command_executor, web_search, web_fetch
agents:
  - name: "MiTest"                   # Agent 名称，唯一标识
    description: "测试Agent"         # Agent 功能描述
    instruction: "你是MiTest，测试Agent。"  # 系统提示词，定义 Agent 的行为和角色
    model: "sonnet"                  # 使用的模型后端名称
    tools:                           # 分配给该 Agent 的工具列表
      - file_reader

# ── HTTP 服务 ──
gin:
  mode: "debug"                      # Gin 运行模式：debug（详细日志）/ release（生产）/ test
  port: 15555                        # HTTP 监听端口，端口被占用时自动递增 10 重试（上限 18000）
  cors:
    allow_origins: ["*"]             # 允许的跨域来源，["*"] 表示允许所有
    allow_methods: ["POST","GET"]    # 允许的 HTTP 方法

# ── CLI / TUI 客户端 ──
# 终端用户界面（TUI）的配置，仅在 chat 模式和默认模式下生效
cli:
  host: "127.0.0.1"                  # 连接的服务端地址
  port: 15555                        # 连接的服务端端口
  lip:                               # 终端颜色方案（支持十六进制颜色）
    base:                            # 基础配色
      foreground: "#00ff11"          # 默认前景色
      background: "#2c2c2cff"        # 默认背景色（含 Alpha 通道）
    title:                           # 标题颜色
      foreground: "#00D787"
    content:                         # AI 回复内容颜色
      foreground: "#FFB86C"
    err:                             # 错误消息颜色
      foreground: "#FF5555"
    help:                            # 帮助信息颜色
      foreground: "#8BE9FD"
    think:                           # 思考状态提示颜色
      foreground: "#FFB86C"
    scroll:                          # 滚动条颜色
      foreground: "#666666"
    separator:                       # 消息分隔线颜色
      foreground: "#444444"
    sidebar:                         # 侧边栏默认文字颜色
      foreground: "#555555"
    sidebar_active:                  # 侧边栏活跃状态颜色
      foreground: "#00D787"
    sidebar_completed:               # 侧边栏已完成状态颜色
      foreground: "#666666"
    sidebar_separator:               # 侧边栏分隔线颜色
      foreground: "#444444"
    sidebar_placeholder:             # 侧边栏占位文字颜色
      foreground: "#555555"
  tui:
    max_history: 100                 # 输入历史最大保存条数（↑↓ 键浏览）
    completable_commands:            # Tab 补全命令列表（输入 / 后按 Tab 自动补全）
      - command: "/help"             # 显示帮助信息
        description: "显示帮助信息"
      - command: "/exit"             # 退出程序
        description: "退出程序"
      - command: "/quit"             # 退出程序（同 /exit）
        description: "退出程序"
      - command: "/viewmemory"       # 查看和切换对话记忆
        description: "查看/切换对话记忆"
      - command: "/excmem"           # 切换到指定的历史会话
        description: "切换到指定记忆"
      - command: "/reback"           # 回退对话到指定的历史轮次
        description: "回退对话到历史轮次"
      - command: "/clear"            # 清除当前对话，创建新会话
        description: "创建并切换到新会话"
      - command: "/prompt"           # 查看或修改 AI 系统提示词
        description: "查看/修改系统提示词"
      - command: "/reload"           # 不重启程序，热重载配置文件和 AI 模型
        description: "热重载配置与模型"
      - command: "/config"           # 在外部编辑器中打开本配置文件
        description: "编辑配置文件"
      - command: "/plan"             # 列出或查看项目计划文件
        description: "列出/编写项目计划"
      - command: "/init"             # 让 AI 分析项目并生成 .mifer/MIFER.md 提示词
        description: "生成项目MIFER.md提示词"
      - command: "/mcp"              # 显示 MCP Server 连接状态
        description: "显示MCP Server状态"
      - command: "/skill"            # 显示已加载的技能列表
        description: "显示已加载的技能列表"
      - command: "/agents"           # 显示已配置的 Agent 列表及其状态
        description: "显示已配置的Agent列表"
      - command: "/compact"          # 手动触发上下文压缩
        description: "手动压缩当前对话上下文"
    content_margin: 4                # 消息内容左右留白列数
    min_height: 10                   # 终端最小行数，低于此值显示"窗口太小"提示
    editor: ""                       # 配置文件编辑器，为空时自动检测（$VISUAL > $EDITOR > 系统默认）
    spinner_type: "MiniDot"          # 等待动画样式（MiniDot / Line / Pulse / Moon / Jump 等）
    spinner_frames: []               # 自定义动画帧，非空时覆盖 spinner_type
    spinner_fps: 0                   # 动画帧率，0 使用 Bubble Tea 默认值
    sidebar_max_log: 100             # 侧边栏 Agent 状态日志最大保留行数
    sidebar_show_tokens: true        # 侧边栏底部是否显示 Token 用量统计
    sidebar_show_timing: true        # 侧边栏是否显示每条日志的时间戳
    completion_max_visible: 5        # Tab 补全弹出列表最多显示的条目数
    mouse_wheel_delta: 3             # 鼠标滚轮每次滚动的行数
    horizontal_scroll_step: 4        # 水平方向每次滚动的列数（Alt+← →）
`
