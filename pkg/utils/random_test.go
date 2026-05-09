package utils

import (
	"testing"
)

func TestPseudoRandom_Deterministic(t *testing.T) {
	input := []byte("test_input")
	result1 := PseudoRandom(input)
	result2 := PseudoRandom(input)
	if result1 != result2 {
		t.Fatalf("相同输入应产生相同输出: got %q and %q", result1, result2)
	}
}

func TestPseudoRandom_DifferentInputs(t *testing.T) {
	a := PseudoRandom([]byte("hello"))
	b := PseudoRandom([]byte("world"))
	if a == b {
		t.Fatal("不同输入应产生不同输出")
	}
}

func TestPseudoRandom_EmptyInput(t *testing.T) {
	result := PseudoRandom([]byte{})
	if result == "" {
		t.Fatal("空输入不应返回空字符串")
	}
}

func TestRandomStr(t *testing.T) {
	s, err := RandomStr(16)
	if err != nil {
		t.Fatalf("RandomStr 失败: %v", err)
	}
	if len(s) != 16 {
		t.Fatalf("期望长度 16, 实际 %d", len(s))
	}
}
