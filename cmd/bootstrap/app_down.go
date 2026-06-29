package bootstrap

import (
	"context"
	"mifer/pkg/logger"
)

func (a *Application) Shutdown(ctx context.Context) error {
	logger.Info("应用正在关闭...")

	// 关闭 HTTP 服务
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			logger.Error("服务器关闭失败", logger.C(err))
		}
	}

	// 释放路由资源（MCP 子进程、确认存储 actor 等）
	if a.Router != nil {
		a.Router.Close()
	}

	logger.Info("应用已关闭")
	return nil
}
