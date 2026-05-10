package bootstrap

import (
	"fmt"
	"mifer/pkg/logger"
	"net/http"
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

	if err = app.initCli(); err != nil {
		return nil, fmt.Errorf("初始化CLI失败: %w", err)
	}
	return app, nil
}

// Run 运行应用
func (a *Application) Run() error {
	a.printStartupInfo()

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Config.Gin.Port),
		Handler: a.Engine,
	}

	logger.Info("HTTP 服务器启动", logger.I("port", a.Config.Gin.Port))
	err := a.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("服务器运行失败: %w", err)
	}
	return nil
}

// printStartupInfo 打印启动信息
func (a *Application) printStartupInfo() {
	logger.Info("应用初始化成功！")
	logger.Info("配置环境:", logger.S("env", a.Config.Env))
	logger.Info("Gin 模式:", logger.S("mode", a.Config.Gin.Mode))
	logger.Info("服务端口:", logger.I("port", a.Config.Gin.Port))
}
