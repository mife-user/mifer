package rag

import (
	"context"

	"mifer/pkg/logger"
)

// Close 释放 Qdrant 连接等资源
func (s *Service) Close() error {
	if s.qdrantClient != nil {
		if err := s.qdrantClient.Close(); err != nil {
			logger.Warn(context.Background(), "关闭Qdrant连接失败", logger.C(err))
			return err
		}
		logger.Debug(context.Background(), "Qdrant连接已关闭")
	}
	return nil
}

// Close 释放 RAG 服务资源（懒加载版本）。
// 若底层 Service 尚未初始化（Qdrant 从未连接），直接返回 nil。
func (s *LazyService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.svc != nil {
		return s.svc.Close()
	}
	return nil
}
