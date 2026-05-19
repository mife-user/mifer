package errorer

import "errors"

const (
	ErrChatTimeout          = "ctx killed or timeout"
	ErrCallBackNull         = "AI 未生成回复内容"
	ErrArgUnknowid          = "未知的参数ID"
	ErrPathCannotCreate     = "路径创建失败"
	ErrIdEmpty              = "ID 不能为空"
	ErrNoBackendConfig      = "未配置任何模型后端，请在配置中添加 ai.backends"
	ErrDefaultBackendConfig = "默认后端未配置，请在 ai.backends 中添加 default"
	ErrApiKey               = "apikey未配置"
)

// New 创建一个新的错误
func New(err string) error {
	return errors.New(err)
}
