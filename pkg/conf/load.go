package conf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// 加载配置文件
func LoadConfig() (*Config, error) {
	var path string
	var fileName string
	v := viper.New()
	//设置默认环境为dev
	v.SetDefault("env", "dev")
	//显式绑定环境变量
	v.BindEnv("env", "MIFER_ENV")
	v.BindEnv("redis.host", "MIFER_REDIS_HOST")
	v.BindEnv("jwt.secret", "MIFER_JWT_SECRET")
	v.BindEnv("redis.password", "MIFER_REDIS_PASSWORD")
	v.BindEnv("ai.base_url", "MIFER_AI_BASEURL")
	v.BindEnv("ai.model", "MIFER_AI_MODEL")
	v.BindEnv("ai.api_key", "MIFER_AI_APIKEY")

	//主要配置文件目录
	env := v.GetString("env")
	if env == "dev" {
		path = "./config"
		fileName = "dev"
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败：%w", err)
		}
		path = filepath.Join(home, "/mifer/config")
		fileName = "prod"
	}

	//创建默认配置文件
	err := newDefaultCfg(env)
	if err != nil {
		return nil, fmt.Errorf("创建默认配置失败：%w", err)
	}

	//添加配置文件路径
	v.AddConfigPath(path)

	//配置文件名称和类型
	v.SetConfigName(fileName)
	v.SetConfigType("yaml")

	//读取主配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("加载主配置失败：%w", err)
	}
	// 手动应用环境变量覆盖（Viper 的 Unmarshal 不经过 BindEnv，需显式覆盖）
	applyEnvOverrides(v)
	//配置到结构体
	if err := v.Unmarshal(&globalConfig); err != nil {
		return nil, fmt.Errorf("解析配置失败：%w", err)
	}

	//设置工作目录
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("获取当前工作目录失败：%w", err)
	}
	globalConfig.Workdir = wd
	return &globalConfig, nil
}

// applyEnvOverrides 手动将环境变量值覆盖到 viper 中。
// Viper 的 Unmarshal 不走 BindEnv 的懒加载路径，必须显式 Set 才能被 Unmarshal 识别。
func applyEnvOverrides(v *viper.Viper) {
	overrides := map[string]string{
		"env":            "MIFER_ENV",
		"redis.host":     "MIFER_REDIS_HOST",
		"redis.password": "MIFER_REDIS_PASSWORD",
		"jwt.secret":     "MIFER_JWT_SECRET",
		"ai.base_url":    "MIFER_AI_BASEURL",
		"ai.model":       "MIFER_AI_MODEL",
		"ai.api_key":     "MIFER_AI_APIKEY",
	}
	for key, envVar := range overrides {
		if val := os.Getenv(envVar); val != "" {
			v.Set(key, val)
		}
	}
}
