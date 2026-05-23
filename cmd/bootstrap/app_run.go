package bootstrap

import (
	"context"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"net/http"
)

// NewApplication 创建应用实例
func NewApplication(ctx context.Context) (*Application, error) {
	var err error
	app := &Application{}

	if err = app.loadConfig(); err != nil {
		return nil, errorer.NewS(errorer.ErrLoadConfigFailed, err)
	}

	if err = app.initontext(ctx); err != nil {
		return nil, errorer.NewS(errorer.ErrInitContextFailed, err)
	}

	if err = app.initLogger(); err != nil {
		return nil, errorer.NewS(errorer.ErrInitLoggerFailed, err)
	}

	// if err = app.initRedis(); err != nil {
	// 	return nil, errorer.NewS(errorer.ErrInitRedisFailed, err)
	// }

	if err = app.initRouter(); err != nil {
		return nil, errorer.NewS(errorer.ErrInitRouterFailed, err)
	}

	if err = app.initCli(); err != nil {
		return nil, errorer.NewS(errorer.ErrInitCLIFailed, err)
	}
	return app, nil
}

// Run 运行应用
func (a *Application) Run() error {
	var err error
	for conf.GetConfig().Gin.Port <= 18000 {
		a.printStartupInfo()
		a.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", conf.GetConfig().Gin.Port),
			Handler: a.Engine,
		}
		logger.Info("HTTP 服务器启动", logger.I("port", conf.GetConfig().Gin.Port))
		err = a.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			conf.GetConfig().Gin.Port += 10
			continue
		}
		return nil
	}
	return errorer.NewS(errorer.ErrServerRunFailed, err)
}

// printStartupInfo 打印启动信息
func (a *Application) printStartupInfo() {
	logger.Info("应用初始化成功！")
	logger.Info("配置环境:", logger.S("env", conf.GetConfig().Env))
	logger.Info("Gin 模式:", logger.S("mode", conf.GetConfig().Gin.Mode))
	logger.Info("服务端口:", logger.I("port", conf.GetConfig().Gin.Port))
}
