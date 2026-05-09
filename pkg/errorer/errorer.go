package errorer

import "errors"

const (
	ErrChatTimeout = "ctx killed or timeout"
)

// New 创建一个新的错误
func New(err string) error {
	return errors.New(err)
}
