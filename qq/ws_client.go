package qq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mifer/pkg/logger"

	"github.com/gorilla/websocket"
)

// wsClient NapCat / SnowLuma WebSocket 客户端。
// 读取：readPump goroutine → eventCh
// 写入：writeCh → writePump goroutine → WebSocket
// Go 原则：通过 channel 通信共享内存，不用锁。
type wsClient struct {
	url     string
	token   string
	eventCh chan *oneBotEvent  // 读事件推送到此 channel
	writeCh chan []byte        // 写数据发送到此 channel
	dialer  *websocket.Dialer
	ctx     context.Context
	cancel  context.CancelFunc
}

// newWSClient 创建 WebSocket 客户端。
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
		writeCh: make(chan []byte, 32),
		dialer:  websocket.DefaultDialer,
	}
}

// WriteJSON 将 v 序列化为 JSON 并发送到写入 channel。
// 若 WS 未连接，数据暂存 channel（buf=32），连接恢复后自动发送。
func (w *wsClient) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case w.writeCh <- data:
	default:
		logger.Warn("QQ WS 写 channel 已满，丢弃消息")
	}
	return nil
}

// connect 连接到 OneBot WebSocket 服务端，阻塞直到 Stop。
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

		// 启动写 goroutine，读 goroutine 在 readPump 中阻塞
		writeCtx, writeCancel := context.WithCancel(w.ctx)
		go w.writePump(writeCtx, conn)

		w.readPump(conn)

		// 读结束（连接断开），通知写 goroutine 退出
		writeCancel()
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

// writePump 从 writeCh 读取数据并写入 WebSocket 连接。
func (w *wsClient) writePump(ctx context.Context, conn *websocket.Conn) {
	logger.Debug("QQ WS writePump 启动")
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-w.writeCh:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Warn("QQ WS 写入失败", logger.C(err))
				return
			}
		}
	}
}

// readPump 从 WebSocket 连接读取事件，推送到 eventCh。
func (w *wsClient) readPump(conn *websocket.Conn) {
	logger.Debug("QQ WS readPump 启动")
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			logger.Warn("QQ WebSocket 读取失败", logger.C(err))
			return
		}

		logger.Debug("QQ WS 收到消息", logger.I("len", len(raw)))

		var event oneBotEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			logger.Warn("QQ WS JSON 解析失败", logger.C(err))
			continue
		}

		// 仅推送 post_type=message，meta_event 和 notice 静默消费
		if event.PostType != "message" {
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
	logger.Debug("QQ WS 客户端停止")
	w.cancel()
}
