package agentconfig

import (
	"os"
	"path/filepath"
	"strings"

	"mifer/pkg/logger"

	"gopkg.in/yaml.v3"
)

// LoadAgents 扫描 ~/.mifer/agents/ 目录，加载所有 .yaml 配置文件。
// 单个文件解析失败时记录日志并跳过，不阻塞其他文件的加载。
// agentsDir 不存在时返回空切片，不报错。
func LoadAgents() ([]*CustomAgentConfig, error) {
	agentsDir, err := resolveAgentDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var configs []*CustomAgentConfig
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(agentsDir, entry.Name())

		cfg, err := loadAgentFile(filePath)
		if err != nil {
			logger.Error("加载自定义Agent配置失败: "+entry.Name(), logger.C(err))
			continue
		}

		if err := cfg.Validate(); err != nil {
			logger.Error("自定义Agent配置校验失败: "+entry.Name(), logger.C(err))
			continue
		}

		// 检测同名冲突（先到先得）
		if seen[cfg.Name] {
			logger.Warn("自定义Agent名称冲突，跳过: "+entry.Name(),
				logger.S("name", cfg.Name),
				logger.S("file", entry.Name()))
			continue
		}
		seen[cfg.Name] = true

		configs = append(configs, cfg)
	}

	return configs, nil
}

// loadAgentFile 解析单个 agent YAML 文件并应用默认值
func loadAgentFile(path string) (*CustomAgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg CustomAgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// resolveAgentDir 返回 agents 配置目录的绝对路径
func resolveAgentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mifer", "agents"), nil
}
