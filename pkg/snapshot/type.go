package snapshot

// FileEntry 描述 changes.jsonl 中单条文件变更记录。
// Hash 为空字符串时表示该文件已在对应轮次被删除。
type FileEntry struct {
	Path  string `json:"path"`  // 相对于工作目录的文件路径
	Hash  string `json:"hash"`  // SHA256 哈希值（hex 编码，64 字符），空字符串表示已删除
	Size  int64  `json:"size"`  // 文件大小（字节），用于快速变更检测
	Mtime int64  `json:"mtime"` // 修改时间（Unix 秒），用于快速变更检测
	Round int    `json:"round"` // 对话轮次
}

// Service 管理基于内容寻址的文件快照。
// 当 baseDir 为空时表示功能禁用，所有操作均为空操作（优雅降级）。
type Service struct {
	workdir      string              // 项目工作目录
	baseDir      string              // 快照存储根目录（空表示禁用）
	skipDirs     map[string]bool     // 快照复制时跳过的目录名
	lastManifest map[string]FileEntry // 每个文件的最新变更条目（内存缓存），key 为相对路径
}
