package memory

import (
	"testing"
)

func TestInit_Memory(t *testing.T) {
	t.Skip("需要有效的配置和上下文，留给集成测试")
}

func TestAppendUser(t *testing.T) {
	m := &Memory{}
	m.AppendUser("你好")
	if len(m.Messages) != 1 {
		t.Fatalf("期望 1 条消息, 实际 %d", len(m.Messages))
	}
	if m.Messages[0].Role != "user" {
		t.Fatalf("期望 role=user, 实际 %q", m.Messages[0].Role)
	}
}

func TestAppendAssistant(t *testing.T) {
	m := &Memory{}
	m.AppendAssistant("你好，有什么需要帮助的？")
	if len(m.Messages) != 1 {
		t.Fatalf("期望 1 条消息, 实际 %d", len(m.Messages))
	}
	if m.Messages[0].Role != "assistant" {
		t.Fatalf("期望 role=assistant, 实际 %q", m.Messages[0].Role)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := MemCfg{MemPath: dir, id: "test123"}
	m := &Memory{Cfg: cfg}
	m.AppendUser("测试消息")
	m.AppendAssistant("回复消息")

	if err := m.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	loaded, err := load(&cfg)
	if err != nil {
		t.Fatalf("load 失败: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("期望 2 条消息, 实际 %d", len(loaded))
	}
	if loaded[0].Content != "测试消息" {
		t.Fatalf("期望 %q, 实际 %q", "测试消息", loaded[0].Content)
	}
	if loaded[1].Content != "回复消息" {
		t.Fatalf("期望 %q, 实际 %q", "回复消息", loaded[1].Content)
	}
}
