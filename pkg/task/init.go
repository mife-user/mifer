package task

import "context"

// Do 在 context 仍有效时执行 task。
// 若 context 已取消则直接返回其错误，否则正常执行 task 并返回结果。
func Do(c context.Context, task func() error) error {
	if err := c.Err(); err != nil {
		return err
	}
	return task()
}
