package qq

import (
	"context"
	"time"

	"mifer/pkg/logger"

	"github.com/gorilla/websocket"
)

// wsClient SnowLuma WebSocket 事件读取器。
// 连接 OneBot WebSocket 服务端，读取事件推送到 eventCh，断线自动重连。
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
	return &wsClient{
		url:     url,
		token:   token,
		eventCh: make(chan *oneBotEvent, 64),
		dialer:  websocket.DefaultDialer,
	}
}

// connect 连接到 OneBot WebSocket 服务端，阻塞读取事件。
// 断线后指数退避重连（1s → 2s → 4s → ... → 60s）。
func (w *wsClient) connect() error {
	delay := 1 * time.Second
	const maxDelay = 60 * time.Second

	for {
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}

		conn, _, err := w.dialer.Dial(w.url, nil)
		if err != nil {
			logger.Warn("QQ WebSocket 连接失败，将重试", logger.S("url", w.url), logger.C(err))
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
		delay = 1 * time.Second // 连接成功，重置重连间隔

		w.readPump(conn)
		conn.Close()

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
	for {
		var event oneBotEvent
		if err := conn.ReadJSON(&event); err != nil {
			logger.Warn("QQ WebSocket 读取失败", logger.C(err))
			return
		}

		// meta_event（heartbeat / lifecycle）静默消费
		if event.PostType == "meta_event" {
			continue
		}

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
	w.cancel()
}
