package errorer

import (
	"errors"
	"strings"
	"testing"
)

// ============================================================================
// errorer_test.go — errorer 包单元测试
// ============================================================================

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"普通错误消息", "something went wrong"},
		{"空字符串", ""},
		{"中文消息", "发生了一个错误"},
		{"带特殊字符", "error: 404 not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.input)
			if err == nil {
				t.Fatal("New() 返回 nil，期望非 nil error")
			}
			if err.Error() != tt.input {
				t.Errorf("New(%q).Error() = %q; want %q", tt.input, err.Error(), tt.input)
			}
		})
	}
}

func TestNewS(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		err    error
		want   string
	}{
		{
			name:   "包装普通错误",
			prefix: "操作失败",
			err:    errors.New("connection refused"),
			want:   "操作失败: connection refused",
		},
		{
			name:   "包装 nil 错误",
			prefix: "操作失败",
			err:    nil,
			want:   "操作失败: %!w(<nil>)", // fmt.Errorf 中 %w 遇到 nil 的实际输出
		},
		{
			name:   "空前缀",
			prefix: "",
			err:    errors.New("some error"),
			want:   ": some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewS(tt.prefix, tt.err)
			if err == nil {
				t.Fatal("NewS() 返回 nil，期望非 nil error")
			}
			if err.Error() != tt.want {
				t.Errorf("NewS(%q, %v).Error() = %q; want %q", tt.prefix, tt.err, err.Error(), tt.want)
			}
		})
	}
}

func TestNewS_Unwrap(t *testing.T) {
	base := errors.New("base error")
	wrapped := NewS("prefix", base)

	// 验证 errors.Is 能匹配到原始错误
	if !errors.Is(wrapped, base) {
		t.Error("NewS 包装的错误应能通过 errors.Is 匹配到原始错误")
	}
}

func TestNewF(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{
			name:   "简单格式化",
			format: "value is %d",
			args:   []any{42},
			want:   "value is 42",
		},
		{
			name:   "多参数",
			format: "%s: %d errors found in %s",
			args:   []any{"build", 3, "main.go"},
			want:   "build: 3 errors found in main.go",
		},
		{
			name:   "无参数",
			format: "static message",
			args:   nil,
			want:   "static message",
		},
		{
			name:   "空格式",
			format: "",
			args:   nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewF(tt.format, tt.args...)
			if err == nil {
				t.Fatal("NewF() 返回 nil，期望非 nil error")
			}
			if err.Error() != tt.want {
				t.Errorf("NewF(%q, %v).Error() = %q; want %q", tt.format, tt.args, err.Error(), tt.want)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	// 验证所有错误常量为非空字符串
	constants := []string{
		ErrChatTimeout,
		ErrCallBackNull,
		ErrNoBackendConfig,
		ErrNoBackendAvailable,
		ErrApiKey,
		ErrTokenInvalid,
		ErrIdEmpty,
		ErrIDIllegalChars,
		ErrToolUnknown,
		ErrConfirmTimeout,
		ErrConfirmDone,
	}

	for _, c := range constants {
		if strings.TrimSpace(c) == "" {
			t.Errorf("错误常量不应为空")
		}
	}
}
