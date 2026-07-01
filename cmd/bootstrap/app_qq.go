package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"mifer/internal/ai/tools/qq"
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

	// 1. 创建 OneBot HTTP 消息发送器，注入到工具包
	qq.Sender = &onebotSender{
		httpURL: cfg.Onebot.HttpURL,
		token:   cfg.Onebot.AccessToken,
	}

	// 2. 创建 QQ adapter（HTTP 消费者，不依赖 internal）
	miferURL := fmt.Sprintf("http://127.0.0.1:%d", conf.GetConfig().Gin.Port)
	app.qqAdapter = qqclient.NewAdapter(qqclient.Config{
		WsURL:          cfg.Onebot.WsURL,
		MiferURL:       miferURL,
		OnebotHttpURL:  cfg.Onebot.HttpURL,
		OnebotToken:    cfg.Onebot.AccessToken,
		BotQQ:          cfg.Bot.QQ,
		GroupReplyMode: cfg.Bot.GroupReplyMode,
		PrivateEnabled: cfg.Bot.PrivateEnabled,
	})

	go func() {
		if err := app.qqAdapter.Start(); err != nil {
			logger.Error("QQ Bot 运行异常", logger.C(err))
		}
	}()

	logger.Info("QQ Bot 已启动", logger.I("qq", int(cfg.Bot.QQ)), logger.S("ws", cfg.Onebot.WsURL))
	return nil
}

// onebotSender 实现 qq.Sender 接口，直接 HTTP 调用 OneBot API。
// 供 internal/ai/tools/qq 包的 qq_send_message 工具调用。
type onebotSender struct {
	httpURL string
	token   string
}

func (s *onebotSender) SendPrivateMsg(userID int64, message string) error {
	return s.call("send_private_msg", map[string]interface{}{
		"user_id":     userID,
		"message":     message,
		"auto_escape": false,
	})
}

func (s *onebotSender) SendGroupMsg(groupID int64, message string) error {
	return s.call("send_group_msg", map[string]interface{}{
		"group_id":    groupID,
		"message":     message,
		"auto_escape": false,
	})
}

func (s *onebotSender) call(action string, params map[string]interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{
		"action": action,
		"params": params,
	})
	url := s.httpURL + "/" + action
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建 OneBot 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("OneBot API 调用失败: %w", err)
	}
	defer resp.Body.Close()
	return nil
}
