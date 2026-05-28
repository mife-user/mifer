package tools

import "context"

type sseCallbackKey struct{}

// WithSSECallback 将 SSE 回调函数注入 context，供 callback handler 使用
func WithSSECallback(ctx context.Context, fn func(event, data string) error) context.Context {
	return context.WithValue(ctx, sseCallbackKey{}, fn)
}

// GetSSECallback 从 context 中获取 SSE 回调函数
func GetSSECallback(ctx context.Context) func(event, data string) error {
	if fn, ok := ctx.Value(sseCallbackKey{}).(func(string, string) error); ok {
		return fn
	}
	return nil
}
