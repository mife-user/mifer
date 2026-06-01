package conf

import (
	"os"
	"path/filepath"
	"slices"

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

// AddToAllowList 添加命令到 .mifer/allowlist.yaml 白名单文件。
// 若文件或目录不存在则自动创建，已存在的命令不重复添加。
func AddToAllowList(workdir, command string) error {
	miferDir := filepath.Join(workdir, ".mifer")
	if err := os.MkdirAll(miferDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(miferDir, "allowlist.yaml")

	// 读取现有列表
	existing, _ := LoadAllowList(workdir)

	// 检查命令是否已存在
	if slices.Contains(existing, command) {
		return nil // 已存在，不重复添加
	}

	// 追加新命令
	cfg := allowListConfig{
		AllowList: append(existing, command),
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.WriteConfigAs(path); err != nil {
		// 文件可能不存在，直接写入
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// 简单写入 YAML 格式
	if _, err := f.WriteString("# 命令执行白名单 — Allow 后自动添加，可手动编辑\n"); err != nil {
		return err
	}
	if _, err := f.WriteString("allow_list:\n"); err != nil {
		return err
	}
	for _, cmd := range cfg.AllowList {
		if _, err := f.WriteString("  - " + cmd + "\n"); err != nil {
			return err
		}
	}

	return nil
}
