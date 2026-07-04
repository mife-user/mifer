package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ============================================================================
// task_test.go — task 包单元测试
// ============================================================================

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	called := false

	err := Do(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Do() 意外错误: %v", err)
	}
	if !called {
		t.Error("任务应被执行但未被调用")
	}
}

func TestDo_TaskError(t *testing.T) {
	ctx := context.Background()
	taskErr := errors.New("任务执行失败")

	err := Do(ctx, func() error {
		return taskErr
	})

	if err == nil {
		t.Fatal("Do() 应返回任务错误，但返回 nil")
	}
	if !errors.Is(err, taskErr) {
		t.Errorf("Do() error = %v; want %v", err, taskErr)
	}
}

func TestDo_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	called := false
	err := Do(ctx, func() error {
		called = true
		return nil
	})

	if err == nil {
		t.Fatal("Do() 在已取消的上下文中应返回错误")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Do() error = %v; want context.Canceled", err)
	}
	if called {
		t.Error("已取消的上下文中任务不应被执行")
	}
}

func TestDo_TimeoutContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // 确保超时

	called := false
	err := Do(ctx, func() error {
		called = true
		return nil
	})

	if err == nil {
		t.Fatal("Do() 在超时上下文中应返回错误")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Do() error = %v; want context.DeadlineExceeded", err)
	}
	if called {
		t.Error("超时上下文中任务不应被执行")
	}
}

func TestDo_TODOContext(t *testing.T) {
	// context.TODO() 是一个非 nil 的占位 context，任务应正常执行
	called := false
	err := Do(context.TODO(), func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("Do(context.TODO()) 意外错误: %v", err)
	}
	if !called {
		t.Error("context.TODO() 下任务应被执行")
	}
}
