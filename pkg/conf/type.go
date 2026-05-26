package conf

var globalConfig Config

// 配置结构体
type Config struct {
	Env  string     `mapstructure:"env"`
	Log  LogConfig  `mapstructure:"log"`
	Gin  GinConfig  `mapstructure:"gin"`
	JWT  JWTConfig  `mapstructure:"jwt"`
	Ai   AiConfig   `mapstructure:"ai"`
	Path PathConfig `mapstructure:"path"`
	Cli  CliConfig  `mapstructure:"cli"`
	Rag  RAGConfig  `mapstructure:"rag"`
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

type TuiConfig struct {
	MaxHistory          int      `mapstructure:"max_history"`
	CompletableCommands []string `mapstructure:"completable_commands"`
	ContentMargin       int      `mapstructure:"content_margin"`
	MinHeight           int      `mapstructure:"min_height"`
	SpinnerType         string   `mapstructure:"spinner_type"`   // 预置类型名，或为空使用自定义帧
	SpinnerFrames       []string `mapstructure:"spinner_frames"` // 自定义动画帧序列，如 [".", "..", "..."]，非空时覆盖 spinner_type
	SpinnerFPS          int      `mapstructure:"spinner_fps"`    // 自定义帧率（帧/秒），默认 10
	SidebarMaxLog       int      `mapstructure:"sidebar_max_log"`     // 状态日志最大行数，默认 100
	SidebarShowTokens   bool     `mapstructure:"sidebar_show_tokens"` // 是否显示token统计，默认 true
	SidebarShowTiming   bool     `mapstructure:"sidebar_show_timing"` // 是否显示时间戳，默认 true
	CompletionMaxVisible  int  `mapstructure:"completion_max_visible"`  // 补全列表最大可见行数，默认 5
	MouseWheelDelta       int  `mapstructure:"mouse_wheel_delta"`       // 垂直滚轮每次滚动行数，默认 3
	HorizontalScrollStep  int  `mapstructure:"horizontal_scroll_step"`  // 水平滚动每次列数，默认 4
}

type Colorlip struct {
	Background string `mapstructure:"background"`
	Foreground string `mapstructure:"foreground"`
}

// 路径配置结构体
type PathConfig struct {
	Workdir string `mapstructure:"workdir"`
	CfgPath string `mapstructure:"cfg_path"`
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
	Backends  map[string]BackendConfig `mapstructure:"backends"`
	AllowList []string                 `mapstructure:"allow_list"` // 命令执行白名单，非空时仅允许白名单内的命令
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

// RAG配置结构体
type RAGConfig struct {
	ChunkSize        int    `mapstructure:"chunk_size"`         // 分块大小（字符数），默认 500
	ChunkOverlap     int    `mapstructure:"chunk_overlap"`      // 重叠字符数，默认 50
	TopK             int    `mapstructure:"top_k"`              // 检索返回数量，默认 5
	Dim              int    `mapstructure:"dim"`                // 向量维度，默认 768（nomic-embed-text）
	QdrantHost       string `mapstructure:"qdrant_host"`        // Qdrant 服务地址，默认 "localhost"
	QdrantPort       int    `mapstructure:"qdrant_port"`        // Qdrant gRPC 端口，默认 6334
	QdrantCollection string `mapstructure:"qdrant_collection"`  // Qdrant 集合名，默认 "mifer_docs"
	QdrantAPIKey     string `mapstructure:"qdrant_api_key"`     // Qdrant API Key（可选）
}
