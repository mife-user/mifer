package qq

import (
	"context"
	"encoding/json"
	"time"

	"mifer/pkg/logger"

	"github.com/gorilla/websocket"
)

// wsClient SnowLuma / NapCat WebSocket 事件读取器。
type wsClient struct {
	url     string
	token   string
	eventCh chan *oneBotEvent
	dialer  *websocket.Dialer
	ctx     context.Context
	cancel  context.CancelFunc
}

// newWSClient 创建 WebSocket 客户端。
func newWSClient(url, token string) *wsClient {
	logger.Debug("QQ 创建 WS 客户端", logger.S("url", url))
	return &wsClient{
		url:     url,
		token:   token,
		eventCh: make(chan *oneBotEvent, 64),
		dialer:  websocket.DefaultDialer,
	}
}

// connect 连接到 OneBot WebSocket 服务端，阻塞读取事件。
func (w *wsClient) connect() error {
	delay := 1 * time.Second
	const maxDelay = 60 * time.Second

	for {
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}

		logger.Info("QQ 正在连接 WebSocket...", logger.S("url", w.url))
		conn, _, err := w.dialer.Dial(w.url, nil)
		if err != nil {
			logger.Warn("QQ WebSocket 连接失败，将重试",
				logger.S("url", w.url),
				logger.S("delay", delay.String()),
				logger.C(err),
			)
			select {
			case <-w.ctx.Done():
				return w.ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		logger.Info("QQ WebSocket 已连接", logger.S("url", w.url))
		delay = 1 * time.Second

		w.readPump(conn)
		conn.Close()

		logger.Warn("QQ WebSocket 断开，将重连", logger.S("delay", delay.String()))
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

// readPump 从 WebSocket 连接读取事件。
func (w *wsClient) readPump(conn *websocket.Conn) {
	logger.Debug("QQ WS readPump 启动")
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			logger.Warn("QQ WebSocket 读取失败", logger.C(err))
			return
		}

		// 打印原始 JSON，便于排查格式问题
		logger.Info("QQ WS 原始消息", logger.S("raw", string(raw)))

		var event oneBotEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			logger.Warn("QQ WS JSON 解析失败", logger.C(err), logger.S("raw", string(raw)))
			continue
		}

		if event.PostType == "meta_event" {
			continue
		}

		logger.Debug("QQ WS 推送事件到 channel",
			logger.S("post_type", event.PostType),
			logger.S("msg_type", event.MessageType),
			logger.I("user", int(event.UserID)),
		)

		select {
		case w.eventCh <- &event:
		case <-w.ctx.Done():
			return
		}
	}
}

// events 返回事件通道。
func (w *wsClient) events() <-chan *oneBotEvent {
	return w.eventCh
}

// stop 停止 WebSocket 客户端。
func (w *wsClient) stop() {
	logger.Debug("QQ WS 客户端停止")
	w.cancel()
}
