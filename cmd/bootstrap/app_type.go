package bootstrap

import (
	"context"
	"net/http"

	"mifer/cli"
	"mifer/internal/api/routes"
	"mifer/qq"

	"github.com/gin-gonic/gin"
)

// Application 应用实例，统一管理所有顶层资源。
// 初始化顺序：loadConfig → initontext → initLogger → initRouter → initCli。
type Application struct {
	Context   context.Context
	Engine    *gin.Engine
	Clier     *cli.Cli
	server    *http.Server
	router    *routes.Router
	qqAdapter *qq.QQAdapter
}
