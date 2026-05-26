package conf

import (
	"mifer/pkg/errorer"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// 加载配置文件
func LoadConfig() (*Config, error) {
	var path string
	var fileName string
	var cfgPath string
	var err error
	v := viper.New()
	//设置默认环境为dev
	v.SetDefault("env", "dev")
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
		path = filepath.Join(home, "/mifer/config")
		cfgPath = filepath.Join(home, "/mifer")
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
	// 后端模型配置 — 支持 MIFER_AI_<BACKEND>_<FIELD> 格式
	for _, backend := range []string{"DEFAULT", "MULTI", "HAIKU", "SONNET", "OPUS"} {
		envPrefix := "MIFER_AI_" + backend + "_"
		if val := os.Getenv(envPrefix + "APIKEY"); val != "" {
			v.Set("ai.backends."+backend+".api_key", val)
		}
		if val := os.Getenv(envPrefix + "BASE_URL"); val != "" {
			v.Set("ai.backends."+backend+".base_url", val)
		}
		if val := os.Getenv(envPrefix + "PROVIDER"); val != "" {
			v.Set("ai.backends."+backend+".provider", val)
		}
		if val := os.Getenv(envPrefix + "MODEL"); val != "" {
			v.Set("ai.backends."+backend+".model", val)
		}
	}
}
