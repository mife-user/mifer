package question

import "context"

// WithStore 将问题存储注入 context。
func WithStore(ctx context.Context, store *Store) context.Context {
	return context.WithValue(ctx, storeKey{}, store)
}

// GetStore 从 context 获取问题存储，可能为 nil。
func GetStore(ctx context.Context) *Store {
	if v := ctx.Value(storeKey{}); v != nil {
		return v.(*Store)
	}
	return nil
}

// WithCallback 将 executor 回调注入 context。
func WithCallback(ctx context.Context, cb ExecutorCallback) context.Context {
	return context.WithValue(ctx, ctxKey{}, cb)
}

// GetCallback 从 context 获取 executor 回调，可能为 nil。
func GetCallback(ctx context.Context) ExecutorCallback {
	if v := ctx.Value(ctxKey{}); v != nil {
		return v.(ExecutorCallback)
	}
	return nil
}

// WithSessionID 将 session ID 注入 context。
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, id)
}

// GetSessionID 从 context 获取 session ID。
func GetSessionID(ctx context.Context) string {
	if v := ctx.Value(sessionIDKey{}); v != nil {
		return v.(string)
	}
	return ""
}
