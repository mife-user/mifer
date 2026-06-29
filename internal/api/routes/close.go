package routes

import "mifer/pkg/logger"

// Close 释放路由持有的所有后台资源（MCP 子进程、确认存储 actor 等）。
// 应在应用关闭时调用。
func (r *Router) Close() {
	logger.Debug("关闭路由资源")
	if r.agentHandler != nil {
		r.agentHandler.CloseExecutor()
	}
}
