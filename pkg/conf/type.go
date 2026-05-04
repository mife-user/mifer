package conf

var globalConfig Config

// 配置结构体
type Config struct {
	Env   string      `mapstructure:"env"`
	Log   LogConfig   `mapstructure:"log"`
	Redis RedisConfig `mapstructure:"redis"`
	Gin   GinConfig   `mapstructure:"gin"`
	JWT   JWTConfig   `mapstructure:"jwt"`
	Ai    AiConfig    `mapstructure:"ai"`
}

// 日志配置结构体
type LogConfig struct {
	MaxSize    int `mapstructure:"max_size"`    // 单文件最大大小（MB），默认 10
	MaxBackups int `mapstructure:"max_backups"` // 保留备份文件最大数量，默认 10
}

// ai配置结构体
type AiConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
	ApiKey  string `mapstructure:"api_key"`
}

// redis配置结构体
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
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
