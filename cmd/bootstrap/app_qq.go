package bootstrap

import (
	"fmt"

	"mifer/pkg/conf"
	"mifer/pkg/logger"
	qqclient "mifer/qq"
)

// initQQ 初始化 QQ Bot，在 initRouter 之后调用。
func (app *Application) initQQ() error {
	cfg := conf.GetConfig().QQ
	if !cfg.Enabled {
		logger.Info("QQ Bot 未启用（qq.enabled=false），跳过初始化")
		return nil
	}
	if cfg.Bot.QQ == 0 {
		logger.Warn("QQ Bot 已启用但未配置 qq 号（qq.bot.qq=0），跳过初始化。" +
			"请在配置文件中填写你的 Bot QQ 号。")
		return nil
	}

	allowedTools := map[string]bool{
		"qq_send_message": true,
	}

	// 创建 QQ adapter（HTTP 消费者，不依赖 internal）
	miferURL := fmt.Sprintf("http://127.0.0.1:%d", conf.GetConfig().Gin.Port)
	app.qqAdapter = qqclient.NewAdapter(qqclient.Config{
		WsURL:          cfg.Onebot.WsURL,
		MiferURL:       miferURL,
		OnebotHttpURL:  cfg.Onebot.HttpURL,
		OnebotToken:    cfg.Onebot.AccessToken,
		BotQQ:          cfg.Bot.QQ,
		GroupReplyMode: cfg.Bot.GroupReplyMode,
		PrivateEnabled: cfg.Bot.PrivateEnabled,
		AllowedTools:   allowedTools,
	})

	go func() {
		if err := app.qqAdapter.Start(); err != nil {
			logger.Error("QQ Bot 运行异常", logger.C(err))
		}
	}()

	logger.Info("QQ Bot 已启动", logger.I("qq", int(cfg.Bot.QQ)), logger.S("ws", cfg.Onebot.WsURL))
	return nil
}
