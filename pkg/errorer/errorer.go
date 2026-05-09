package errorer

import "errors"

const (
	ErrChatTimeout  = "ctx killed or timeout"
	ErrCallBackNull = "AI 未生成回复内容"
)

// New 创建一个新的错误
func New(err string) error {
	return errors.New(err)
}
