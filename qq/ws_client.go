package qq

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"mifer/pkg/logger"

	"github.com/gorilla/websocket"
)

// wsClient NapCat / SnowLuma WebSocket 客户端。
// 同时处理事件读取（readPump）和动作写入（WriteJSON）。
type wsClient struct {
	url     string
	token   string
	eventCh chan *oneBotEvent
	dialer  *websocket.Dialer
	ctx     context.Context
	cancel  context.CancelFunc

	// 写保护：同一连接不能并发写入
	conn   *websocket.Conn
	writeMu sync.Mutex
}

// newWSClient 创建 WebSocket 客户端。
// 若 token 非空，拼接到 URL 查询参数 ?access_token=xxx。
func newWSClient(rawURL, token string) *wsClient {
	if token != "" {
		if strings.Contains(rawURL, "?") {
			rawURL += "&access_token=" + token
		} else {
			rawURL += "?access_token=" + token
		}
	}
	logger.Debug("QQ 创建 WS 客户端", logger.S("url", rawURL))
	return &wsClient{
		url:     rawURL,
		token:   token,
		eventCh: make(chan *oneBotEvent, 64),
		dialer:  websocket.DefaultDialer,
	}
}

// WriteJSON 向 WebSocket 连接写入 JSON 消息（如 OneBot action）。
// 线程安全，可与 readPump 并发使用。
func (w *wsClient) WriteJSON(v interface{}) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	if w.conn == nil {
		return nil // 静默失败，等待重连
	}
	return w.conn.WriteJSON(v)
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

		// 保存连接引用，供 onebotClient 发送消息
		w.writeMu.Lock()
		w.conn = conn
		w.writeMu.Unlock()

		w.readPump(conn)

		// 连接断开
		w.writeMu.Lock()
		w.conn = nil
		w.writeMu.Unlock()

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

		logger.Debug("QQ WS 原始消息", logger.I("len", len(raw)))

		var event oneBotEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			logger.Warn("QQ WS JSON 解析失败", logger.C(err))
			continue
		}

		if event.PostType == "meta_event" {
			continue
		}

		// 只推送 post_type=message 的事件，notice 静默消费
		if event.PostType != "message" {
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
