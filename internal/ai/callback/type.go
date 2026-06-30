package callback

// executorCallback executor 层向上传递事件的标准回调签名。
type executorCallback func(event, content string) error

type ctxKey struct{}
