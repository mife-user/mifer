package errorer

import "errors"

const ()

// New 创建一个新的错误
func New(err string) error {
	return errors.New(err)
}
