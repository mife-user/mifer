package bootstrap

import (
	"fmt"
	"mifer/internal/api/routes"
)

// initRouter 初始化路由
func (a *Application) initRouter() error {
	router := routes.GetRouter()
	if !router.NewRouter(a.Context, a.Config) {
		return fmt.Errorf("创建路由失败")
	}
	a.Engine = router.Setup()
	return nil
}
