package exc

import (
	"testing"
)

// ============================================================================
// exc_test.go — exc 包单元测试
// ============================================================================

func TestIsUint(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected uint
		ok       bool
	}{
		{"正整数", uint(42), 42, true},
		{"零值", uint(0), 0, true},
		{"大数", uint(4294967295), 4294967295, true},
		{"int 类型", int(42), 0, false},
		{"string 类型", "42", 0, false},
		{"nil 值", nil, 0, false},
		{"float 类型", float64(3.14), 0, false},
		{"bool 类型", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := IsUint(tt.input)
			if ok != tt.ok {
				t.Errorf("IsUint(%v) ok = %v; want %v", tt.input, ok, tt.ok)
			}
			if ok && got != tt.expected {
				t.Errorf("IsUint(%v) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		ok       bool
	}{
		{"普通字符串", "hello", "hello", true},
		{"空字符串", "", "", true},
		{"中文字符串", "你好世界", "你好世界", true},
		{"int 类型", int(42), "", false},
		{"uint 类型", uint(42), "", false},
		{"nil 值", nil, "", false},
		{"[]byte 类型", []byte("hello"), "", false},
		{"bool 类型", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := IsString(tt.input)
			if ok != tt.ok {
				t.Errorf("IsString(%v) ok = %v; want %v", tt.input, ok, tt.ok)
			}
			if ok && got != tt.expected {
				t.Errorf("IsString(%v) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStrToUint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint
		wantErr bool
	}{
		{"零", "0", 0, false},
		{"正整数", "42", 42, false},
		{"最大 uint32", "4294967295", 4294967295, false},
		{"空字符串", "", 0, true},
		{"负数", "-1", 0, true},
		{"浮点数", "3.14", 0, true},
		{"非数字", "abc", 0, true},
		{"超出 uint32 范围", "4294967296", 0, true},
		{"含空格", " 42 ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StrToUint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrToUint(%q) error = %v; wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StrToUint(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestUintToStr(t *testing.T) {
	tests := []struct {
		name  string
		input uint
		want  string
	}{
		{"零", 0, "0"},
		{"正整数", 42, "42"},
		{"大数", 4294967295, "4294967295"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UintToStr(tt.input)
			if err != nil {
				t.Errorf("UintToStr(%d) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("UintToStr(%d) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStrToUint_UintToStr_Roundtrip(t *testing.T) {
	// 往返测试：StrToUint → UintToStr 应还原
	inputs := []string{"0", "1", "42", "100", "4294967295"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			u, err := StrToUint(input)
			if err != nil {
				t.Fatalf("StrToUint(%q) unexpected error: %v", input, err)
			}
			s, err := UintToStr(u)
			if err != nil {
				t.Fatalf("UintToStr(%d) unexpected error: %v", u, err)
			}
			if s != input {
				t.Errorf("往返失败: %q → %d → %q", input, u, s)
			}
		})
	}
}

func TestExcFileToJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "简单结构体",
			input: struct{ Name string }{"test"},
			want:  `{"Name":"test"}`,
		},
		{
			name:  "空结构体",
			input: struct{}{},
			want:  `{}`,
		},
		{
			name:  "map 类型",
			input: map[string]int{"a": 1, "b": 2},
			want:  `{"a":1,"b":2}`,
		},
		{
			name:  "nil 值",
			input: nil,
			want:  `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExcFileToJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExcFileToJSON(%v) error = %v; wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ExcFileToJSON(%v) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExcJSONToFile(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		target  interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:   "简单结构体",
			json:   `{"Name":"test"}`,
			target: &struct{ Name string }{},
			want:   &struct{ Name string }{Name: "test"},
		},
		{
			name:   "空 JSON 对象",
			json:   `{}`,
			target: &struct{}{},
			want:   &struct{}{},
		},
		{
			name:    "非法 JSON",
			json:    `{invalid}`,
			target:  &struct{}{},
			wantErr: true,
		},
		{
			name:    "空字符串",
			json:    "",
			target:  &struct{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExcJSONToFile(tt.json, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExcJSONToFile(%q) error = %v; wantErr %v", tt.json, err, tt.wantErr)
			}
		})
	}
}

func TestExcFileToJSON_ExcJSONToFile_Roundtrip(t *testing.T) {
	// 往返测试：序列化 → 反序列化 应保持数据一致
	type Person struct {
		Name string
		Age  int
	}
	original := Person{Name: "张三", Age: 25}

	jsonStr, err := ExcFileToJSON(original)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var restored Person
	err = ExcJSONToFile(jsonStr, &restored)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if restored != original {
		t.Errorf("往返失败: %+v → JSON → %+v", original, restored)
	}
}
