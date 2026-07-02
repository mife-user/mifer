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
	WsURL          string          // SnowLuma WebSocket 地址
	MiferURL       string          // Mifer HTTP 服务地址
	OnebotHttpURL  string          // OneBot HTTP API 地址
	OnebotToken    string          // OneBot access_token
	BotQQ          int64           // Bot 自己的 QQ 号
	GroupReplyMode string          // "mention_only" / "always"
	PrivateEnabled bool            // 是否响应私聊
	AllowedTools   map[string]bool // 工具自动确认白名单
}

// IsMentionOnly 是否仅在 @ 了 Bot 时才回复群聊消息。
func (c Config) IsMentionOnly() bool { return c.GroupReplyMode == "mention_only" }

// ─────────────────────────── OneBot 事件 ───────────────────────────

// oneBotEvent OneBot v11 消息事件，仅解析实际使用的字段。
type oneBotEvent struct {
	PostType    string      `json:"post_type"`    // "message" | "meta_event"
	MessageType string      `json:"message_type"` // "private" | "group"
	UserID      int64       `json:"user_id"`
	GroupID     int64       `json:"group_id"`
	Message     string      `json:"message"`     // CQ 码格式
	RawMessage  string      `json:"raw_message"` // 纯文本
	Sender      eventSender `json:"sender"`      // 发送者信息
}

// IsPrivate 是否私聊消息。
func (e *oneBotEvent) IsPrivate() bool { return e.MessageType == "private" }

// IsGroup 是否群聊消息。
func (e *oneBotEvent) IsGroup() bool { return e.MessageType == "group" }

// IsMessage 是否为消息事件（排除 meta_event、notice 等）。
func (e *oneBotEvent) IsMessage() bool { return e.PostType == "message" }

type eventSender struct {
	Nickname string `json:"nickname"` // 昵称
}

// ─────────────────────────── HTTP 客户端 ───────────────────────────

// newHTTPClient 创建带超时的 HTTP 客户端。
func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

// ─────────────────────────── Sender 接口 ───────────────────────────

// Sender QQ 消息发送器接口，用于 internal/ai/tools/qq 包的函数注入。
// qq/ 包提供 OneBot HTTP 和 WebSocket 两种实现。
type Sender interface {
	SendPrivateMsg(userID int64, message string) error
	SendGroupMsg(groupID int64, message string) error
}
