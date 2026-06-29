package bootstrap

import (
	"mifer/internal/api/routes"
	"mifer/pkg/errorer"
)

// initRouter 初始化路由，router 在 Shutdown 时由 Application 统一释放。
func (a *Application) initRouter() error {
	a.router = routes.GetRouter()
	if err := a.router.NewRouter(a.Context); err != nil {
		return errorer.NewS(errorer.ErrCreateRouterFailed, err)
	}
	a.Engine = a.router.Setup()
	return nil
}
