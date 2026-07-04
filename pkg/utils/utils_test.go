package utils

import (
	"strings"
	"testing"
)

// ============================================================================
// utils_test.go — utils 包单元测试
// ============================================================================

func TestHashPassword_CheckPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"普通密码", "myPassword123"},
		{"短密码", "ab"},
		{"长密码", strings.Repeat("a", 72)}, // bcrypt 最大长度
		{"中文密码", "我的密码123"},
		{"特殊字符", "p@$$w0rd!#%&*()"},
		{"空密码", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword(%q) 意外错误: %v", tt.password, err)
			}
			if hash == "" {
				t.Fatal("HashPassword() 返回空哈希")
			}
			if hash == tt.password {
				t.Error("哈希值不应等于原始密码")
			}

			// 验证正确密码
			if !CheckPassword(tt.password, hash) {
				t.Error("CheckPassword() 对正确密码返回 false")
			}

			// 验证错误密码
			if CheckPassword("wrongPassword", hash) {
				t.Error("CheckPassword() 对错误密码返回 true")
			}
		})
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"空哈希", ""},
		{"非法哈希", "not-a-valid-bcrypt-hash"},
		{"短哈希", "$2a$10$short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if CheckPassword("anything", tt.hash) {
				t.Errorf("CheckPassword(%q, %q) 应对非法哈希返回 false", "anything", tt.hash)
			}
		})
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	// 同一密码两次哈希应产生不同结果（bcrypt 随机加盐）
	hash1, err := HashPassword("samePassword")
	if err != nil {
		t.Fatalf("第一次 HashPassword 失败: %v", err)
	}
	hash2, err := HashPassword("samePassword")
	if err != nil {
		t.Fatalf("第二次 HashPassword 失败: %v", err)
	}
	if hash1 == hash2 {
		t.Error("同一密码两次哈希应产生不同结果（随机加盐）")
	}
	// 但两个哈希都应能验证原始密码
	if !CheckPassword("samePassword", hash1) {
		t.Error("hash1 应能验证原始密码")
	}
	if !CheckPassword("samePassword", hash2) {
		t.Error("hash2 应能验证原始密码")
	}
}

func TestPseudoRandom(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"普通字符串", []byte("hello")},
		{"空输入", []byte{}},
		{"数字输入", []byte("12345")},
		{"二进制数据", []byte{0x00, 0xFF, 0xAB}},
		{"中文输入", []byte("你好世界")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PseudoRandom(tt.input)
			if result == "" {
				t.Error("PseudoRandom() 返回空字符串")
			}
			// 验证结果是纯数字
			for _, c := range result {
				if c < '0' || c > '9' {
					t.Errorf("PseudoRandom() 返回非数字字符: %q", result)
					break
				}
			}
		})
	}
}

func TestPseudoRandom_Deterministic(t *testing.T) {
	// 相同输入应产生相同输出
	input := []byte("test-deterministic")
	result1 := PseudoRandom(input)
	result2 := PseudoRandom(input)
	if result1 != result2 {
		t.Errorf("PseudoRandom() 对相同输入应返回相同结果: %q vs %q", result1, result2)
	}
}

func TestPseudoRandom_DifferentInputs(t *testing.T) {
	// 不同输入应通常产生不同输出（哈希冲突极少见）
	r1 := PseudoRandom([]byte("input-a"))
	r2 := PseudoRandom([]byte("input-b"))
	if r1 == r2 {
		t.Error("不同输入产生相同哈希值（极低概率的哈希冲突）")
	}
}

func TestRandomStr(t *testing.T) {
	lengths := []int{1, 8, 16, 32, 64}
	for _, length := range lengths {
		t.Run(string(rune(length)), func(t *testing.T) {
			s, err := RandomStr(length)
			if err != nil {
				t.Fatalf("RandomStr(%d) 意外错误: %v", length, err)
			}
			if len(s) != length {
				t.Errorf("RandomStr(%d) 长度 = %d; want %d", length, len(s), length)
			}
		})
	}
}

func TestRandomStr_Uniqueness(t *testing.T) {
	// 两次调用应大概率产生不同结果
	s1, err := RandomStr(16)
	if err != nil {
		t.Fatalf("RandomStr 失败: %v", err)
	}
	s2, err := RandomStr(16)
	if err != nil {
		t.Fatalf("RandomStr 失败: %v", err)
	}
	if s1 == s2 {
		t.Error("两次 RandomStr(16) 产生相同结果（极低概率）")
	}
}
