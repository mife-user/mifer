package bootstrap

import (
	"mifer/internal/api/routes"
	"mifer/pkg/errorer"
)

// initRouter 初始化路由
func (a *Application) initRouter() error {
	a.Router = routes.GetRouter()
	if err := a.Router.NewRouter(a.Context); err != nil {
		return errorer.NewS(errorer.ErrCreateRouterFailed, err)
	}
	a.Engine = a.Router.Setup()
	return nil
}
