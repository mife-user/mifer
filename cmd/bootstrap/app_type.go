package bootstrap

import (
	"mifer/pkg/conf"

	"github.com/gin-gonic/gin"
)

type Application struct {
	Config *conf.Config
	Engine *gin.Engine
}
