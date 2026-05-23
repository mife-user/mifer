package bootstrap

import (
	"context"
	"net/http"
	"mifer/cli"

	"github.com/gin-gonic/gin"
)

type Application struct {
	Context context.Context
	Engine  *gin.Engine
	Clier   *cli.Cli
	server  *http.Server
}
