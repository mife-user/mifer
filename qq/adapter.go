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
	return &QQAdapter{
		cfg:    cfg,
		ws:     newWSClient(cfg.WsURL, cfg.OnebotToken),
		mifer:  &miferClient{baseURL: cfg.MiferURL},
		onebot: &onebotClient{httpURL: cfg.OnebotHttpURL, token: cfg.OnebotToken},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动 adapter：连接 SnowLuma WebSocket 并进入事件循环。
// 阻塞直到 Stop() 被调用或连接永久失败，应在 goroutine 中调用。
func (a *QQAdapter) Start() error {
	a.ws.ctx, a.ws.cancel = context.WithCancel(a.ctx)
	go a.eventLoop()
	return a.ws.connect()
}

// Stop 停止 adapter，关闭 WebSocket 连接和事件循环。
func (a *QQAdapter) Stop() {
	a.cancel()
	a.ws.stop()
}

// ─────────────────────────── 事件循环 ───────────────────────────

// eventLoop 从 WebSocket 事件通道读取事件并处理。
func (a *QQAdapter) eventLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case event, ok := <-a.ws.events():
			if !ok {
				return
			}
			a.handleMessage(event)
		}
	}
}

// ─────────────────────────── 消息分发 ───────────────────────────

// handleMessage 按消息类型分发处理。
func (a *QQAdapter) handleMessage(event *oneBotEvent) {
	switch event.MessageType {
	case "private":
		if !a.cfg.PrivateEnabled {
			return
		}
		a.handlePrivate(event)
	case "group":
		if a.cfg.GroupReplyMode == "mention_only" && !isAtBot(event.Message, a.cfg.BotQQ) {
			return
		}
		a.handleGroup(event)
	}
}

// handlePrivate 处理私聊消息。
func (a *QQAdapter) handlePrivate(event *oneBotEvent) {
	query := cleanCQ(event.Message)
	if query == "" {
		return
	}
	sid := buildSessionID(event)
	a.processAndReply(sid, event, query)
}

// handleGroup 处理群聊消息。
func (a *QQAdapter) handleGroup(event *oneBotEvent) {
	query := cleanCQ(event.Message)
	if query == "" {
		return
	}
	nick := event.Sender.Nickname
	if nick == "" {
		nick = fmt.Sprintf("%d", event.UserID)
	}
	context := fmt.Sprintf("[%s]: %s", nick, query)
	sid := buildSessionID(event)
	a.processAndReply(sid, event, context)
}

// ─────────────────────────── 核心流水线 ───────────────────────────

// processAndReply 切换记忆 → Agent 对话 → 发送回复。
func (a *QQAdapter) processAndReply(sessionID string, event *oneBotEvent, query string) {
	// 1. 切换记忆会话
	if err := a.mifer.exchangeMemory(sessionID); err != nil {
		logger.Error("QQ切换记忆失败", logger.S("session", sessionID), logger.C(err))
		return
	}

	// 2. SSE 对话，收集回复并自动处理工具确认
	var replyBuf strings.Builder
	err := a.mifer.chat(query, func(eventType, data string) error {
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

	// 3. 发送回复
	reply := strings.TrimSpace(replyBuf.String())
	if reply != "" {
		a.onebot.sendReply(event, reply)
	}
}

// ─────────────────────────── Session ID ───────────────────────────

// buildSessionID 从 OneBot 事件构建记忆会话 ID。
// 私聊：qq_private/{user_id}
// 群聊：qq_group/{group_id}/{user_id}
func buildSessionID(event *oneBotEvent) string {
	if event.MessageType == "private" {
		return fmt.Sprintf("qq_private/%d", event.UserID)
	}
	return fmt.Sprintf("qq_group/%d/%d", event.GroupID, event.UserID)
}
