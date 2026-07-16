package bootstrap

import (
	"context"
	"mifer/pkg/logger"
)

// Shutdown 优雅关闭应用：先停 HTTP 服务，再释放路由资源（MCP 子进程、确认存储 actor 等）。
func (a *Application) Shutdown(ctx context.Context) error {
	logger.Info(ctx, "应用正在关闭...")

	// 停止接受新请求
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			logger.Error(ctx, "服务器关闭失败", logger.C(err))
		}
	}

	// 释放路由资源
	if a.router != nil {
		a.router.Close()
	}

	logger.Info(ctx, "应用已关闭")
	return nil
}
