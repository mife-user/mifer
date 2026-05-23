package rag

// Close 关闭 Redis 连接
func (s *Service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
