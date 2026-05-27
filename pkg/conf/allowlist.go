package conf

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// allowListConfig 白名单配置文件内部结构体
type allowListConfig struct {
	AllowList []string `mapstructure:"allow_list"`
}

// LoadAllowList 从工作目录下的 .mifer/allowlist.yaml 加载命令白名单
// 文件不存在时返回空列表（不报错），表示不启用白名单检查
func LoadAllowList(workdir string) ([]string, error) {
	path := filepath.Join(workdir, ".mifer", "allowlist.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg allowListConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return cfg.AllowList, nil
}
