package rag

import "mifer/pkg/conf"

// RAGConfig 引用 conf 包中的统一定义，与 YAML 配置文件 mapstructure 绑定
type RAGConfig = conf.RAGConfig

// DefaultRAGConfig 从全局配置读取 RAG 参数，零值字段使用默认值填充
func DefaultRAGConfig() *RAGConfig {
	cfg := conf.GetConfig().Rag
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = 500
	}
	if cfg.ChunkOverlap == 0 {
		cfg.ChunkOverlap = 50
	}
	if cfg.IndexName == "" {
		cfg.IndexName = "mifer_docs"
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "mifer:docs:"
	}
	if cfg.TopK == 0 {
		cfg.TopK = 5
	}
	if cfg.Dim == 0 {
		cfg.Dim = 768
	}
	return &cfg
}
