package errorer

import "errors"

const (
	ErrChatTimeout      = "ctx killed or timeout"
	ErrCallBackNull     = "AI 未生成回复内容"
	ErrArgUnknowid      = "未知的参数ID"
	ErrPathCannotCreate = "路径创建失败"
)

// New 创建一个新的错误
func New(err string) error {
	return errors.New(err)
}
