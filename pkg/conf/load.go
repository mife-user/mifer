package conf

import (
	"mifer/pkg/errorer"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// 加载配置文件
func LoadConfig() (*Config, error) {
	var path string
	var fileName string
	var cfgPath string
	var err error
	v := viper.New()
	//设置默认环境为prod
	v.SetDefault("env", "prod")
	// 手动应用环境变量覆盖（Viper 的 Unmarshal 不经过 BindEnv，需显式覆盖）
	applyEnvOverrides(v)
	//主要配置文件目录
	env := v.GetString("env")
	if env == "dev" {
		path = "./config"
		fileName = "dev"
		cfgPath, err = os.Getwd()
		if err != nil {
			return nil, errorer.NewS(errorer.ErrGetWorkDirFailed, err)
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, errorer.NewS(errorer.ErrGetHomeDirFailed, err)
		}
		path = filepath.Join(home, "/.mifer/config")
		cfgPath = filepath.Join(home, "/.mifer")
		fileName = "prod"
	}
	//设置默认路径
	v.Set("path.cfg_path", cfgPath)

	//创建默认配置文件
	err = newDefaultCfg(env)
	if err != nil {
		return nil, errorer.NewS(errorer.ErrCreateDefaultConfigFailed, err)
	}

	//添加配置文件路径
	v.AddConfigPath(path)

	//配置文件名称和类型
	v.SetConfigName(fileName)
	v.SetConfigType("yaml")

	//读取主配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, errorer.NewS(errorer.ErrLoadMainConfigFailed, err)
	}

	//配置到结构体
	if err := v.Unmarshal(&globalConfig); err != nil {
		return nil, errorer.NewS(errorer.ErrParseConfigFailed, err)
	}

	//设置工作目录
	wd, err := os.Getwd()
	if err != nil {
		return nil, errorer.NewS(errorer.ErrGetWorkDirFailed, err)
	}
	globalConfig.Path.Workdir = wd
	return &globalConfig, nil
}

// applyEnvOverrides 手动将环境变量值覆盖到 viper 中。
// Viper 的 Unmarshal 不走 BindEnv 的懒加载路径，必须显式 Set 才能被 Unmarshal 识别。
func applyEnvOverrides(v *viper.Viper) {
	overrides := map[string]string{
		"env":        "MIFER_ENV",
		"jwt.secret": "MIFER_JWT_SECRET",
	}
	// 基础字段
	for key, envVar := range overrides {
		if val := os.Getenv(envVar); val != "" {
			v.Set(key, val)
		}
	}
	// RAG 配置
	ragOverrides := map[string]string{
		"rag.chunk_size":        "MIFER_RAG_CHUNK_SIZE",
		"rag.chunk_overlap":     "MIFER_RAG_CHUNK_OVERLAP",
		"rag.top_k":             "MIFER_RAG_TOP_K",
		"rag.qdrant_host":       "MIFER_RAG_QDRANT_HOST",
		"rag.qdrant_port":       "MIFER_RAG_QDRANT_PORT",
		"rag.qdrant_collection": "MIFER_RAG_QDRANT_COLLECTION",
	}
	for key, envVar := range ragOverrides {
		if val := os.Getenv(envVar); val != "" {
			v.Set(key, val)
		}
	}
	// Web Search 配置
	searchOverrides := map[string]string{
		"search.provider": "MIFER_SEARCH_PROVIDER",
		"search.api_key":  "MIFER_SEARCH_API_KEY",
		"search.base_url": "MIFER_SEARCH_BASE_URL",
	}
	for key, envVar := range searchOverrides {
		if val := os.Getenv(envVar); val != "" {
			v.Set(key, val)
		}
	}
	// 后端 API Key 覆盖 — 支持 MIFER_AI_BACKENDS_<NAME>_APIKEY 格式
	// 敏感信息通过环境变量传入，避免写入配置文件
	prefix := "MIFER_AI_BACKENDS_"
	suffix := "_APIKEY"
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) || !strings.Contains(env, suffix+"=") {
			continue
		}
		// 格式：MIFER_AI_BACKENDS_<NAME>_APIKEY=<VALUE>
		rest := env[len(prefix):]
		eqIdx := strings.IndexByte(rest, '=')
		if eqIdx < 0 {
			continue
		}
		keyPart := rest[:eqIdx] // <NAME>_APIKEY
		// 去掉末尾的 _APIKEY
		if !strings.HasSuffix(keyPart, "_APIKEY") {
			continue
		}
		backendName := keyPart[:len(keyPart)-7] // 去掉 "_APIKEY"
		val := rest[eqIdx+1:]
		v.Set("ai.backends."+backendName+".api_key", val)
	}
}
