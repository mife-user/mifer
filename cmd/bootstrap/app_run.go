package bootstrap

import (
	"fmt"
	"mifer/pkg/logger"
)

// NewApplication 创建应用实例
func NewApplication() (*Application, error) {
	var err error
	app := &Application{}

	if err = app.loadConfig(); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	if err = app.initontext(); err != nil {
		return nil, fmt.Errorf("初始化上下文失败: %w", err)
	}

	if err = app.initLogger(); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	// if err = app.initRedis(); err != nil {
	// 	return nil, fmt.Errorf("初始化Redis失败: %w", err)
	// }

	if err = app.initRouter(); err != nil {
		return nil, fmt.Errorf("初始化路由失败: %w", err)
	}

	return app, nil
}

// Run 运行应用
func (a *Application) Run() error {
	a.printStartupInfo()
	return a.Engine.Run(fmt.Sprintf(":%d", a.Config.Gin.Port))
}

// printStartupInfo 打印启动信息
func (a *Application) printStartupInfo() {
	logger.Info("应用初始化成功！")
	logger.Info("配置环境:", logger.S("env", a.Config.Env))
	logger.Info("Gin 模式:", logger.S("mode", a.Config.Gin.Mode))
	logger.Info("服务端口:", logger.I("port", a.Config.Gin.Port))
}
