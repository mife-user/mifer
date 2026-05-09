package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("hello123")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if hash == "" {
		t.Fatal("hash 为空")
	}
	if hash == "hello123" {
		t.Fatal("hash 不应与原文相同")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "secure_password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Fatal("正确密码验证应通过")
	}
	if CheckPassword("wrong_password", hash) {
		t.Fatal("错误密码验证应失败")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("空密码哈希不应报错: %v", err)
	}
	if hash == "" {
		t.Fatal("空密码 hash 不应为空")
	}
}
