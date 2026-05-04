package task

import "context"

func Do(c context.Context, task func() error) error {
	done := make(chan struct{}, 1)
	var err error = nil
	go func() {
		defer close(done)
		err = task()
	}()
	for {
		select {
		case <-c.Done():
			return c.Err()
		case <-done:
			return err
		}
	}
}
