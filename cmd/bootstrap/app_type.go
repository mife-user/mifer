package bootstrap

import (
	"context"
	"net/http"
	"mifer/cli"
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
)

type Application struct {
	Context context.Context
	Config  *conf.Config
	Engine  *gin.Engine
	Clier   *cli.Cli
	server  *http.Server
}
