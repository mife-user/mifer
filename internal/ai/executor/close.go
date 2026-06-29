package executor

import "mifer/pkg/logger"

// Close 释放 Executor 持有的所有资源（MCP 子进程、确认存储 actor 等）。
// 调用后 Executor 不应再被使用。
func (e *Executor) Close() {
	if e.Humen == nil {
		return
	}
	if e.Humen.MCPManager != nil {
		logger.Debug("关闭 MCP Manager")
		e.Humen.MCPManager.Close()
	}
	if e.Humen.ConfirmStore != nil {
		logger.Debug("关闭确认存储")
		e.Humen.ConfirmStore.Close()
	}
	logger.Debug("Executor 资源已释放")
}
