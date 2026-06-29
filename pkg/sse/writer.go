// Package sse 提供 SSE（Server-Sent Events）写入基础设施，
// 通过单 goroutine 序列化所有写入操作，内置心跳保活机制。
package sse

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// FlushWriter SSE 写入所需的最小接口。
// gin.ResponseWriter、net/http.ResponseWriter（含 Flusher）均满足。
type FlushWriter interface {
	io.Writer
	Flush()
}

// sseMsg 是一条 SSE 写入任务。
type sseMsg struct {
	write func(w FlushWriter) error
	errCh chan<- error
}

// Writer 管理 SSE 写入队列和心跳。
// 所有写入通过 [Writer.SendSync] 或 [Writer.SendFire] 提交，
// 由内部 writer goroutine 串行执行，无需外部加锁。
type Writer struct {
	w      FlushWriter
	cancel context.CancelFunc
	msgCh  chan sseMsg
	closed atomic.Bool // writer goroutine 退出后置为 true，防止 SendSync 死锁
}

// New 创建 Writer 并启动 writer goroutine 和心跳 goroutine。
// cancel 在写入失败（连接断开）时由 Writer 调用，调用方应将其与 context.WithCancel 配合使用。
func New(ctx context.Context, w FlushWriter, heartbeat time.Duration, cancel context.CancelFunc) *Writer {
	sw := &Writer{
		w:      w,
		cancel: cancel,
		msgCh:  make(chan sseMsg, 16),
	}

	// writer goroutine：独占 FlushWriter，串行执行所有写入
	go func() {
		defer sw.closed.Store(true)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sw.msgCh:
				if !ok {
					return
				}
				if ctx.Err() != nil {
					if msg.errCh != nil {
						msg.errCh <- ctx.Err()
					}
					continue
				}
				err := msg.write(sw.w)
				if msg.errCh != nil {
					msg.errCh <- err
				}
				if err != nil {
					cancel()
				}
			}
		}
	}()

	// 心跳 goroutine
	go func() {
		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sw.SendFire(func(w FlushWriter) error {
					_, err := fmt.Fprintf(w, ": heartbeat\n\n")
					w.Flush()
					return err
				})
			}
		}
	}()

	return sw
}

// SendSync 提交同步写入任务，阻塞等待 writer goroutine 完成并返回错误。
// 若写入失败或 writer 已退出，返回 context.Canceled。
func (sw *Writer) SendSync(write func(w FlushWriter) error) error {
	// writer goroutine 已退出时直接返回，防止死锁
	if sw.closed.Load() {
		return context.Canceled
	}

	errCh := make(chan error, 1)
	select {
	case sw.msgCh <- sseMsg{write: write, errCh: errCh}:
	default:
		// channel 满说明 writer goroutine 卡死（如 TCP 半开连接），不应阻塞调用方
		return context.Canceled
	}

	// writer goroutine 可能在入队后退出，需二次检查防止死锁
	if sw.closed.Load() {
		return context.Canceled
	}

	if err := <-errCh; err != nil {
		sw.cancel()
		return context.Canceled
	}
	return nil
}

// SendFire 提交 fire-and-forget 写入任务。writer 退出或 channel 满时静默丢弃。
func (sw *Writer) SendFire(write func(w FlushWriter) error) {
	if sw.closed.Load() {
		return
	}
	select {
	case sw.msgCh <- sseMsg{write: write, errCh: nil}:
	default:
	}
}
