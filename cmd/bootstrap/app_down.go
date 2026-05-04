package bootstrap

import (
	"context"
	"mifer/pkg/logger"
)

// Shutdown 关闭应用
func (a *Application) Shutdown(ctx context.Context) error {
	logger.Info("应用正在关闭...")
	logger.Info("应用已关闭")
	return nil
}
