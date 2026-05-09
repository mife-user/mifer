package task

import "context"

func Do(c context.Context, task func() error) error {
	select {
	case <-c.Done():
		return c.Err()
	default:
		err := task()
		if err != nil {
			return err
		}
	}
	return nil
}
