package bootstrap

import (
	"fmt"
	"mifer/internal/api/routes"
)

// initRouter 初始化路由
func (a *Application) initRouter() error {
	router := routes.GetRouter()
	if err := router.NewRouter(a.Context, a.Config); err != nil {
		return fmt.Errorf("创建路由失败: %w", err)
	}
	a.Engine = router.Setup()
	return nil
}
