// Package offload 提供工具输出结果的卸载存储接口，支持将大体积工具结果从上下文
// 中移出并保存到外部存储，替换为占位符引用，从而控制上下文长度。
package offload

import "context"

// Offloader 定义 tool 结果卸载存储的抽象接口。
// 默认实现为 LocalOffloader（本地文件系统），可通过实现此接口扩展为 S3、Redis 等后端。
type Offloader interface {
	// Save 将 content 以 key 为标识保存到后端，返回可供后续检索的文件路径。
	Save(ctx context.Context, key string, content []byte) (filePath string, err error)
	// Load 根据 filePath 从后端读取已保存的内容。
	Load(ctx context.Context, filePath string) ([]byte, error)
	// Delete 根据 filePath 从后端删除已保存的内容。
	Delete(ctx context.Context, filePath string) error
}
