package rag

// Close 释放 Milvus 连接等资源（当前索引器和检索器由外部管理生命周期）
func (s *Service) Close() error {
	return nil
}
