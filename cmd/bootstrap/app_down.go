package bootstrap

import (
	"context"
	"mifer/pkg/logger"
)

func (a *Application) Shutdown(ctx context.Context) error {
	logger.Info("应用正在关闭...")
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			logger.Error("服务器关闭失败", logger.C(err))
		}
	}
	logger.Info("应用已关闭")
	return nil
}
