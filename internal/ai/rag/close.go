package rag

import "mifer/pkg/logger"

// Close 释放 Qdrant 连接等资源
func (s *Service) Close() error {
	if s.qdrantClient != nil {
		if err := s.qdrantClient.Close(); err != nil {
			logger.Warn("关闭Qdrant连接失败", logger.C(err))
			return err
		}
		logger.Debug("Qdrant连接已关闭")
	}
	return nil
}
