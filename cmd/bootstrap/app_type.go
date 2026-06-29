package bootstrap

import (
	"context"
	"net/http"

	"mifer/cli"
	"mifer/internal/api/routes"

	"github.com/gin-gonic/gin"
)

type Application struct {
	Context context.Context
	Engine  *gin.Engine
	Clier   *cli.Cli
	server  *http.Server
	Router  *routes.Router
}
