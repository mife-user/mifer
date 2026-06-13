package confirm

import (
	"encoding/json"
	"slices"

	"mifer/pkg/conf"
	"mifer/pkg/logger"
)

// NeedConfirm 返回一个函数，判断指定工具是否需要确认。
// 逻辑：enabled 且不在 exclude 列表且不在 session 白名单中。
// command_executor 额外检查 allowlist。
func NeedConfirm(store *Store) func(toolName, arguments, sessionID string) bool {
	// 解析 allowlist 列表（服务启动时加载一次，后续可通过 /reload 刷新）
	allowlist := loadAllowlist()

	return func(toolName, arguments, sessionID string) bool {
		cfg := conf.GetConfig().Confirm
		if !cfg.Enabled {
			return false
		}

		// 检查排除列表
		if slices.Contains(cfg.Exclude, toolName) {
			return false
		}

		// 命令行工具：检查全局 allowlist
		if toolName == "command_executor" {
			cmd := parseCommandFromArgs(arguments)
			if slices.Contains(allowlist, cmd) {
				return false
			}
		}

		// Session 白名单（非命令工具 Allow 后生效）
		if store.IsAllowed(sessionID, toolName) {
			return false
		}

		return true
	}
}

// loadAllowlist 加载命令白名单，静默处理文件不存在。
func loadAllowlist() []string {
	list, err := conf.LoadAllowList(conf.GetConfig().Path.Workdir)
	if err != nil {
		logger.Warn("加载白名单配置失败，使用空列表", logger.C(err))
		return nil
	}
	return list
}

// parseCommandFromArgs 从 command_executor 的 JSON 参数中解析实际命令。
// 参数格式: {"command": "dir"} 或 {"command": "dir C:\\path"}
func parseCommandFromArgs(arguments string) string {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return ""
	}
	return args.Command
}
