package qq

import (
	"context"
	"fmt"
	"strings"

	"mifer/pkg/logger"
)

// ─────────────────────────── 构造与生命周期 ───────────────────────────

// NewAdapter 创建 QQ adapter。所有外部依赖通过 Config 注入。
func NewAdapter(cfg Config) *QQAdapter {
	ctx, cancel := context.WithCancel(context.Background())
	ws := newWSClient(cfg.WsURL, cfg.OnebotToken)
	httpCli := newHTTPClient()
	return &QQAdapter{
		cfg:   cfg,
		ws:    ws,
		mifer: &miferClient{
			baseURL:      cfg.MiferURL,
			httpClient:   httpCli,
			allowedTools: cfg.AllowedTools,
		},
		onebot: &onebotClient{ws: ws},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动 adapter：连接 SnowLuma WebSocket 并进入事件循环。
func (a *QQAdapter) Start() error {
	logger.Info("QQ 适配器启动",
		logger.S("ws", a.cfg.WsURL),
		logger.S("mifer", a.cfg.MiferURL),
		logger.S("onebot", a.cfg.OnebotHttpURL),
		logger.I("bot_qq", int(a.cfg.BotQQ)),
		logger.S("group_mode", a.cfg.GroupReplyMode),
	)
	a.ws.ctx, a.ws.cancel = context.WithCancel(a.ctx)
	go a.eventLoop()
	return a.ws.connect()
}

// Stop 停止 adapter。
func (a *QQAdapter) Stop() {
	logger.Info("QQ 适配器停止")
	a.cancel()
	a.ws.stop()
}

// ─────────────────────────── 事件循环 ───────────────────────────

func (a *QQAdapter) eventLoop() {
	logger.Info("QQ 事件循环已启动")
	for {
		select {
		case <-a.ctx.Done():
			logger.Info("QQ 事件循环退出")
			return
		case event, ok := <-a.ws.events():
			if !ok {
				logger.Warn("QQ 事件通道已关闭")
				return
			}
			logger.Debug("QQ 收到事件",
				logger.S("post_type", event.PostType),
				logger.S("message_type", event.MessageType),
				logger.I("user_id", int(event.UserID)),
			)
			a.handleMessage(event)
		}
	}
}

// ─────────────────────────── 消息分发 ───────────────────────────

func (a *QQAdapter) handleMessage(event *oneBotEvent) {
	logger.Info("QQ 收到消息",
		logger.S("type", event.MessageType),
		logger.I("user", int(event.UserID)),
		logger.S("nick", event.Sender.Nickname),
		logger.S("raw", event.RawMessage),
	)

	switch event.MessageType {
	case "private":
		if !a.cfg.PrivateEnabled {
			logger.Debug("QQ 私聊已禁用，跳过")
			return
		}
		a.handlePrivate(event)
	case "group":
		if a.cfg.IsMentionOnly() {
			at := isAtBot(event.Message, a.cfg.BotQQ)
			logger.Debug("QQ 群聊 @ 检测",
				logger.I("group", int(event.GroupID)),
				logger.I("bot_qq", int(a.cfg.BotQQ)),
				logger.S("at_bot", fmt.Sprintf("%v", at)),
			)
			if !at {
				return
			}
		}
		a.handleGroup(event)
	default:
		logger.Debug("QQ 未知消息类型，跳过", logger.S("type", event.MessageType))
	}
}

func (a *QQAdapter) handlePrivate(event *oneBotEvent) {
	query := cleanCQ(event.Message)
	logger.Debug("QQ 私聊清洗后", logger.S("query", query))
	if query == "" {
		return
	}
	sid := buildSessionID(event)
	logger.Info("QQ 处理私聊", logger.S("session", sid), logger.S("query", query))
	a.processAndReply(sid, event, query)
}

func (a *QQAdapter) handleGroup(event *oneBotEvent) {
	query := cleanCQ(event.Message)
	logger.Debug("QQ 群聊清洗后", logger.S("query", query))
	if query == "" {
		return
	}
	nick := event.Sender.Nickname
	if nick == "" {
		nick = fmt.Sprintf("%d", event.UserID)
	}
	ctx := fmt.Sprintf("[%s]: %s", nick, query)
	sid := buildSessionID(event)
	logger.Info("QQ 处理群聊", logger.S("session", sid), logger.S("context", ctx))
	a.processAndReply(sid, event, ctx)
}

// ─────────────────────────── 核心流水线 ───────────────────────────

func (a *QQAdapter) processAndReply(sessionID string, event *oneBotEvent, query string) {
	// SSE 对话（sessionID 传入请求体，由服务端原子完成记忆切换 + 对话）
	logger.Info("QQ 开始对话", logger.S("session", sessionID), logger.S("query", query))
	var replyBuf strings.Builder
	err := a.mifer.chat(sessionID, query, func(eventType, data string) error {
		logger.Debug("QQ SSE事件", logger.S("type", eventType), logger.I("len", len(data)))
		switch eventType {
		case "response":
			replyBuf.WriteString(data)
		case "tool_confirm":
			a.mifer.confirmTool(data)
		}
		return nil
	})
	if err != nil {
		logger.Error("QQ对话失败", logger.S("session", sessionID), logger.C(err))
		a.onebot.sendReply(event, "抱歉，处理消息时出现了错误。")
		return
	}

	// 发送回复
	reply := strings.TrimSpace(replyBuf.String())
	logger.Info("QQ 对话完成", logger.S("session", sessionID), logger.I("reply_len", len(reply)))
	if reply != "" {
		logger.Debug("QQ 发送回复", logger.S("reply", reply))
		a.onebot.sendReply(event, reply)
	} else {
		logger.Warn("QQ 回复为空，不发送")
	}
}

// ─────────────────────────── Session ID ───────────────────────────

func buildSessionID(event *oneBotEvent) string {
	if event.IsPrivate() {
		return fmt.Sprintf("qq_private/%d", event.UserID)
	}
	return fmt.Sprintf("qq_group/%d/%d", event.GroupID, event.UserID)
}
