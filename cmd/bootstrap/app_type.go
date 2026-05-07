package bootstrap

import (
	"context"
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
)

type Application struct {
	Context context.Context
	Config  *conf.Config
	Engine  *gin.Engine
}
