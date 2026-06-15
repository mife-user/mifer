package snapshot

// Service 管理文件快照的创建与恢复。
// 当 baseDir 为空时表示功能禁用，所有操作均为空操作（优雅降级）。
type Service struct {
	workdir string // 项目工作目录
	baseDir string // 快照存储根目录（空表示禁用）
}

// skipDirs 快照复制时跳过的目录名
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".mifer":       true,
	"memory":       true, // 运行时记忆数据（避免快照覆盖自身）
	"config":       true, // 配置文件目录
	"logs":         true, // 日志文件目录
}
