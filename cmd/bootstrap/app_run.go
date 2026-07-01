package bootstrap

import (
	"context"
	"fmt"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"net/http"
	"time"
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
		logger.Error("初始化路由失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrInitRouterFailed, err)
	}

	if err = app.initQQ(); err != nil {
		logger.Warn("QQ Bot 初始化失败（不影响核心服务）", logger.C(err))
	}

	if err = app.initCli(); err != nil {
		logger.Error("初始化CLI失败", logger.C(err))
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
			Addr:         fmt.Sprintf(":%d", conf.GetConfig().Gin.Port),
			Handler:      a.Engine,
			ReadTimeout:  5 * time.Minute,
			WriteTimeout: 0, // SSE 长连接不设写入超时
		}
		logger.Info("HTTP 服务器启动", logger.I("port", conf.GetConfig().Gin.Port))
		err = a.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error("端口绑定失败，切换端口重试", logger.I("port", conf.GetConfig().Gin.Port), logger.C(err))
			conf.GetConfig().Gin.Port += 10
			continue
		}
		return nil
	}
	logger.Error("所有端口均不可用，服务启动失败", logger.C(err))
	return errorer.NewS(errorer.ErrServerRunFailed, err)
}

// printStartupInfo 打印启动信息
func (a *Application) printStartupInfo() {
	logger.Info("应用初始化成功！")
	logger.Info("配置环境:", logger.S("env", conf.GetConfig().Env))
	logger.Info("Gin 模式:", logger.S("mode", conf.GetConfig().Gin.Mode))
	logger.Info("服务端口:", logger.I("port", conf.GetConfig().Gin.Port))
}
