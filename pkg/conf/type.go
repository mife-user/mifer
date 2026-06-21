package conf

var globalConfig Config

// allowListConfig 白名单配置文件内部结构体
type allowListConfig struct {
	AllowList []string `mapstructure:"allow_list"`
}

// 配置结构体
type Config struct {
	Env     string        `mapstructure:"env"`
	Log     LogConfig     `mapstructure:"log"`
	Gin     GinConfig     `mapstructure:"gin"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	Ai      AiConfig      `mapstructure:"ai"`
	Agents  []AgentConfig `mapstructure:"agents"`
	Path    PathConfig    `mapstructure:"path"`
	Cli     CliConfig     `mapstructure:"cli"`
	Rag     RAGConfig     `mapstructure:"rag"`
	Mcp     MCPConfig     `mapstructure:"mcp"`
	Search  SearchConfig  `mapstructure:"search"`
	Skill   SkillConfig   `mapstructure:"skill"`
	Confirm ConfirmConfig `mapstructure:"confirm"`
}

// cli配置结构体
type CliConfig struct {
	Host string    `mapstructure:"host"`
	Port int       `mapstructure:"port"`
	Lip  LipConfig `mapstructure:"lip"`
	Tui  TuiConfig `mapstructure:"tui"`
}

type LipConfig struct {
	Base               Colorlip `mapstructure:"base"`
	Title              Colorlip `mapstructure:"title"`
	Content            Colorlip `mapstructure:"content"`
	Err                Colorlip `mapstructure:"err"`
	Help               Colorlip `mapstructure:"help"`
	Think              Colorlip `mapstructure:"think"`
	Scroll             Colorlip `mapstructure:"scroll"`
	Separator          Colorlip `mapstructure:"separator"`
	Sidebar            Colorlip `mapstructure:"sidebar"`
	SidebarActive      Colorlip `mapstructure:"sidebar_active"`
	SidebarCompleted   Colorlip `mapstructure:"sidebar_completed"`
	SidebarSeparator   Colorlip `mapstructure:"sidebar_separator"`
	SidebarPlaceholder Colorlip `mapstructure:"sidebar_placeholder"`
}

// CompletableCommand 可补全命令定义
type CompletableCommand struct {
	Command     string `mapstructure:"command"`     // 命令名，如 "/help"
	Description string `mapstructure:"description"` // 说明，如 "显示帮助信息"
}

type TuiConfig struct {
	MaxHistory           int                  `mapstructure:"max_history"`
	CompletableCommands  []CompletableCommand `mapstructure:"completable_commands"`
	ContentMargin        int                  `mapstructure:"content_margin"`
	MinHeight            int                  `mapstructure:"min_height"`
	SpinnerType          string               `mapstructure:"spinner_type"`           // 预置类型名，或为空使用自定义帧
	SpinnerFrames        []string             `mapstructure:"spinner_frames"`         // 自定义动画帧序列，如 [".", "..", "..."]，非空时覆盖 spinner_type
	SpinnerFPS           int                  `mapstructure:"spinner_fps"`            // 自定义帧率（帧/秒），默认 10
	SidebarMaxLog        int                  `mapstructure:"sidebar_max_log"`        // 状态日志最大行数，默认 100
	SidebarShowTokens    bool                 `mapstructure:"sidebar_show_tokens"`    // 是否显示token统计，默认 true
	SidebarShowTiming    bool                 `mapstructure:"sidebar_show_timing"`    // 是否显示时间戳，默认 true
	CompletionMaxVisible int                  `mapstructure:"completion_max_visible"` // 补全列表最大可见行数，默认 5
	MouseWheelDelta      int                  `mapstructure:"mouse_wheel_delta"`      // 垂直滚轮每次滚动行数，默认 3
	HorizontalScrollStep int                  `mapstructure:"horizontal_scroll_step"` // 水平滚动每次列数，默认 4

	// 编辑器
	Editor string `mapstructure:"editor"` // 外部编辑器命令（如 "code --wait"），为空时自动检测
}

type Colorlip struct {
	Background string `mapstructure:"background"`
	Foreground string `mapstructure:"foreground"`
}

// 路径配置结构体
type PathConfig struct {
	Workdir         string `mapstructure:"workdir"`
	CfgPath         string `mapstructure:"cfg_path"`
	SnapshotEnabled bool   `mapstructure:"snapshot_enabled"` // 是否启用文件快照（每轮对话后自动保存，reback 时恢复）
}

// 日志配置结构体
type LogConfig struct {
	MaxSize    int    `mapstructure:"max_size"`    // 单文件最大大小（MB），默认 10
	MaxBackups int    `mapstructure:"max_backups"` // 保留备份文件最大数量，默认 10
	Level      string `mapstructure:"level"`       // 日志级别（debug/info/warn/error），为空时由 env 决定
}

// 后端模型配置结构体
type BackendConfig struct {
	Provider string `mapstructure:"provider"` // 模型提供商：openai / claude / gemini / ollama
	BaseURL  string `mapstructure:"base_url"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
}

// ai配置结构体
type AiConfig struct {
	Backends map[string]BackendConfig `mapstructure:"backends"`
	Context  ContextConfig            `mapstructure:"context"` // 上下文压缩配置
}

// agent自定义
type AgentConfig struct {
	Name        string   `mapstructure:"name"`
	Description string   `mapstructure:"description"`
	Instruction string   `mapstructure:"instruction"`
	Model       string   `mapstructure:"model"`
	Tools       []string `mapstructure:"tools"`
}

// gin配置结构体
type GinConfig struct {
	Mode string     `mapstructure:"mode"`
	Port int        `mapstructure:"port"`
	Cors CorsConfig `mapstructure:"cors"`
}

// CORS配置结构体
type CorsConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
}

// JWT配置结构体
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

// MCP配置结构体
type MCPConfig struct {
	Servers []MCPServerConfig `mapstructure:"servers"`
}

// MCPServerConfig 单个 MCP Server 的连接配置
type MCPServerConfig struct {
	Name    string   `mapstructure:"name"`    // 唯一标识，如 "filesystem"
	Command string   `mapstructure:"command"` // 启动命令，如 "npx"
	Args    []string `mapstructure:"args"`    // 参数，如 ["-y", "@anthropic/mcp-server-filesystem", "/path"]
	Env     []string `mapstructure:"env"`     // 环境变量（可选），如 ["GITHUB_TOKEN=xxx"]
	Agents  []string `mapstructure:"agents"`  // 分配给哪些 Agent，空或 ["*"] 表示全部
	Enabled bool     `mapstructure:"enabled"` // 是否启用，默认 true
}

// SearchConfig 网页搜索配置
type SearchConfig struct {
	Provider   string `mapstructure:"provider"`    // 搜索方式：留空默认 searxng / bing（Azure API）/ duckduckgo（国外）
	APIKey     string `mapstructure:"api_key"`     // API密钥（仅 bing provider 需要，Azure 免费层 1000次/月）
	BaseURL    string `mapstructure:"base_url"`    // 自定义搜索地址（searxng 默认 http://localhost:8080）
	MaxResults int    `mapstructure:"max_results"` // 默认最大搜索结果数，默认5
	TimeoutSec int    `mapstructure:"timeout_sec"` // HTTP请求超时秒数，默认10
}

// Skill配置结构体
type SkillConfig struct {
	Path    string `mapstructure:"path"`    // 技能目录路径，为空时自动选择默认路径
	Enabled bool   `mapstructure:"enabled"` // 是否启用，默认 true
}

// ConfirmConfig 工具调用确认配置
type ConfirmConfig struct {
	Enabled    bool     `mapstructure:"enabled"`     // 是否启用工具确认，所有工具调用都需要确认
	TimeoutSec int      `mapstructure:"timeout_sec"` // 确认超时秒数，默认 60
	Exclude    []string `mapstructure:"exclude"`     // 排除不需要确认的工具名列表
}

// ContextConfig 上下文压缩配置
type ContextConfig struct {
	Length       int     `mapstructure:"length"`        // 上下文长度阈值（token数），默认 1000000
	Threshold    float64 `mapstructure:"threshold"`     // 触发压缩的比例，默认 0.8
	Model        string  `mapstructure:"model"`         // 压缩用模型后端名，默认 "haiku"
	RecentRounds int     `mapstructure:"recent_rounds"` // 压缩后保留的最近完整对话轮数，默认 3
}

// RAG配置结构体
type RAGConfig struct {
	ChunkSize        int    `mapstructure:"chunk_size"`        // 分块大小（字符数），默认 500
	ChunkOverlap     int    `mapstructure:"chunk_overlap"`     // 重叠字符数，默认 50
	TopK             int    `mapstructure:"top_k"`             // 检索返回数量，默认 5
	Dim              int    `mapstructure:"dim"`               // 向量维度，默认 768（nomic-embed-text）
	QdrantHost       string `mapstructure:"qdrant_host"`       // Qdrant 服务地址，默认 "localhost"
	QdrantPort       int    `mapstructure:"qdrant_port"`       // Qdrant gRPC 端口，默认 6334
	QdrantCollection string `mapstructure:"qdrant_collection"` // Qdrant 集合名，默认 "mifer_docs"
	QdrantAPIKey     string `mapstructure:"qdrant_api_key"`    // Qdrant API Key（可选）
}
