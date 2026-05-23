package bootstrap

import (
	"mifer/internal/api/routes"
	"mifer/pkg/errorer"
)

// initRouter 初始化路由
func (a *Application) initRouter() error {
	router := routes.GetRouter()
	if err := router.NewRouter(a.Context); err != nil {
		return errorer.NewS(errorer.ErrCreateRouterFailed, err)
	}
	a.Engine = router.Setup()
	return nil
}
