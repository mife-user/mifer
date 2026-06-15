package snapshot

// FileEntry 描述快照清单中单个文件的元信息。
type FileEntry struct {
	Hash  string `json:"hash"`  // SHA256 哈希值（hex 编码，64 字符）
	Size  int64  `json:"size"`  // 文件大小（字节），用于快速变更检测
	Mtime int64  `json:"mtime"` // 修改时间（Unix 秒），用于快速变更检测
}

// Manifest 表示一个快照轮次的完整文件清单：相对路径 → 文件元信息。
type Manifest map[string]FileEntry

// Service 管理基于内容寻址的文件快照。
// 当 baseDir 为空时表示功能禁用，所有操作均为空操作（优雅降级）。
type Service struct {
	workdir      string          // 项目工作目录
	baseDir      string          // 快照存储根目录（空表示禁用）
	skipDirs     map[string]bool // 快照复制时跳过的目录名
	lastManifest Manifest        // 最近一次快照的清单，供 SaveRound 做变更检测（内存缓存）
}
