package qq

import (
	"context"
	"net/http"
	"time"
)

// ─────────────────────────── QQAdapter ───────────────────────────

// QQAdapter QQ 消息通道客户端，通过 SnowLuma（OneBot v11）收发 QQ 消息。
// 作为 Mifer HTTP API 的消费者，不依赖任何 internal/ 包。
type QQAdapter struct {
	cfg    Config
	ws     *wsClient
	mifer  *miferClient
	onebot *onebotClient
	ctx    context.Context
	cancel context.CancelFunc
}

// Config QQ adapter 运行时配置。
type Config struct {
	WsURL          string // SnowLuma WebSocket 地址
	MiferURL       string // Mifer HTTP 服务地址
	OnebotHttpURL  string // OneBot HTTP API 地址
	OnebotToken    string // OneBot access_token
	BotQQ          int64  // Bot 自己的 QQ 号
	GroupReplyMode string // "mention_only" / "always"
	PrivateEnabled bool   // 是否响应私聊
}

// ─────────────────────────── OneBot 事件 ───────────────────────────

// oneBotEvent OneBot v11 消息事件，仅解析实际使用的字段。
type oneBotEvent struct {
	PostType    string      `json:"post_type"`    // "message" | "meta_event"
	MessageType string      `json:"message_type"` // "private" | "group"
	UserID      int64       `json:"user_id"`
	GroupID     int64       `json:"group_id"`
	Message     string      `json:"message"`    // CQ 码格式
	RawMessage  string      `json:"raw_message"` // 纯文本
	Sender      eventSender `json:"sender"`
}

type eventSender struct {
	Nickname string `json:"nickname"`
}

// ─────────────────────────── HTTP 客户端 ───────────────────────────

// httpClient 统一的 HTTP 客户端配置。
var httpClient = &http.Client{Timeout: 60 * time.Second}
