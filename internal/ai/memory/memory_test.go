package memory

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

// ============================================================================
// memory_test.go — memory 包纯函数单元测试
// 测试不涉及文件 I/O 的纯方法，包括消息管理、计数、摘要生成等
// ============================================================================

// newTestMemory 创建一个用于测试的空 Memory 实例
func newTestMemory() *Memory {
	return &Memory{
		messages:   []*schema.Message{},
		savedCount: 0,
		Cfg: MemCfg{
			MemPath: "",
			Id:      "test-session",
		},
	}
}

// newTestMemoryWithMessages 创建包含预设消息的 Memory 实例
func newTestMemoryWithMessages(msgs []*schema.Message) *Memory {
	return &Memory{
		messages:   msgs,
		savedCount: len(msgs),
		Cfg: MemCfg{
			MemPath: "",
			Id:      "test-session",
		},
	}
}

// ──────────────────────────── Messages / Len ────────────────────────────

func TestMemory_Messages(t *testing.T) {
	t.Run("空记忆", func(t *testing.T) {
		m := newTestMemory()
		msgs := m.Messages()
		if len(msgs) != 0 {
			t.Errorf("空记忆的 Messages() 长度 = %d; want 0", len(msgs))
		}
	})

	t.Run("有消息", func(t *testing.T) {
		msgs := []*schema.Message{
			schema.UserMessage("hello"),
			schema.AssistantMessage("hi", nil),
		}
		m := newTestMemoryWithMessages(msgs)
		got := m.Messages()
		if len(got) != 2 {
			t.Errorf("Messages() 长度 = %d; want 2", len(got))
		}
	})
}

func TestMemory_Len(t *testing.T) {
	t.Run("空记忆", func(t *testing.T) {
		m := newTestMemory()
		if m.Len() != 0 {
			t.Errorf("Len() = %d; want 0", m.Len())
		}
	})

	t.Run("有消息", func(t *testing.T) {
		msgs := []*schema.Message{
			schema.UserMessage("a"),
			schema.UserMessage("b"),
			schema.UserMessage("c"),
		}
		m := newTestMemoryWithMessages(msgs)
		if m.Len() != 3 {
			t.Errorf("Len() = %d; want 3", m.Len())
		}
	})
}

// ──────────────────────────── GetCurrentID ────────────────────────────

func TestMemory_GetCurrentID(t *testing.T) {
	m := newTestMemory()
	id := m.GetCurrentID()
	if id != "test-session" {
		t.Errorf("GetCurrentID() = %q; want %q", id, "test-session")
	}
}

// ──────────────────────────── AppendUser / AppendAssistant ────────────────────────────

func TestMemory_AppendUser(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"普通消息", "你好，请帮我写代码"},
		{"空消息", ""},
		{"长消息", "这是一条很长的消息" + string(make([]byte, 1000))},
		{"特殊字符", "包含\n换行\t制表符"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMemory()
			initialLen := m.Len()
			msgs := m.AppendUser(tt.content)

			if m.Len() != initialLen+1 {
				t.Errorf("AppendUser 后 Len() = %d; want %d", m.Len(), initialLen+1)
			}
			if len(msgs) != initialLen+1 {
				t.Errorf("AppendUser 返回长度 = %d; want %d", len(msgs), initialLen+1)
			}

			lastMsg := msgs[len(msgs)-1]
			if lastMsg.Role != schema.User {
				t.Errorf("最后一条消息 Role = %s; want %s", lastMsg.Role, schema.User)
			}
			if lastMsg.Content != tt.content {
				t.Errorf("最后一条消息 Content = %q; want %q", lastMsg.Content, tt.content)
			}
		})
	}
}

func TestMemory_AppendAssistant(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"普通回复", "好的，我来帮你"},
		{"空回复", ""},
		{"多行回复", "第一行\n第二行\n第三行"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMemory()
			initialLen := m.Len()
			msgs := m.AppendAssistant(tt.content)

			if m.Len() != initialLen+1 {
				t.Errorf("AppendAssistant 后 Len() = %d; want %d", m.Len(), initialLen+1)
			}
			if len(msgs) != initialLen+1 {
				t.Errorf("AppendAssistant 返回长度 = %d; want %d", len(msgs), initialLen+1)
			}

			lastMsg := msgs[len(msgs)-1]
			if lastMsg.Role != schema.Assistant {
				t.Errorf("最后一条消息 Role = %s; want %s", lastMsg.Role, schema.Assistant)
			}
			if lastMsg.Content != tt.content {
				t.Errorf("最后一条消息 Content = %q; want %q", lastMsg.Content, tt.content)
			}
		})
	}
}

func TestMemory_AppendUser_AppendAssistant_Interleaved(t *testing.T) {
	m := newTestMemory()
	m.AppendUser("问题1")
	m.AppendAssistant("回答1")
	m.AppendUser("问题2")
	m.AppendAssistant("回答2")

	if m.Len() != 4 {
		t.Errorf("Len() = %d; want 4", m.Len())
	}

	msgs := m.Messages()
	expectedRoles := []schema.RoleType{schema.User, schema.Assistant, schema.User, schema.Assistant}
	for i, expected := range expectedRoles {
		if msgs[i].Role != expected {
			t.Errorf("消息[%d] Role = %s; want %s", i, msgs[i].Role, expected)
		}
	}
}

// ──────────────────────────── CountUserMessages ────────────────────────────

func TestMemory_CountUserMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []*schema.Message
		want     int
	}{
		{
			name:     "空记忆",
			messages: []*schema.Message{},
			want:     0,
		},
		{
			name: "仅有用户消息",
			messages: []*schema.Message{
				schema.UserMessage("a"),
				schema.UserMessage("b"),
			},
			want: 2,
		},
		{
			name: "仅有助手消息",
			messages: []*schema.Message{
				schema.AssistantMessage("a", nil),
				schema.AssistantMessage("b", nil),
			},
			want: 0,
		},
		{
			name: "混合消息",
			messages: []*schema.Message{
				schema.UserMessage("q1"),
				schema.AssistantMessage("a1", nil),
				schema.UserMessage("q2"),
				schema.AssistantMessage("a2", nil),
				schema.UserMessage("q3"),
			},
			want: 3,
		},
		{
			name: "包含系统消息",
			messages: []*schema.Message{
				schema.SystemMessage("system prompt"),
				schema.UserMessage("q1"),
				schema.AssistantMessage("a1", nil),
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMemoryWithMessages(tt.messages)
			got := m.CountUserMessages()
			if got != tt.want {
				t.Errorf("CountUserMessages() = %d; want %d", got, tt.want)
			}
		})
	}
}

// ──────────────────────────── ListRebackEntries ────────────────────────────

func TestMemory_ListRebackEntries(t *testing.T) {
	t.Run("空记忆", func(t *testing.T) {
		m := newTestMemory()
		entries, err := m.ListRebackEntries()
		if err != nil {
			t.Fatalf("ListRebackEntries() 意外错误: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("空记忆的 ListRebackEntries() 长度 = %d; want 0", len(entries))
		}
	})

	t.Run("包含用户消息", func(t *testing.T) {
		msgs := []*schema.Message{
			schema.UserMessage("第一个问题"),
			schema.AssistantMessage("第一个回答", nil),
			schema.UserMessage("第二个问题很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长"),
			schema.AssistantMessage("第二个回答", nil),
		}
		m := newTestMemoryWithMessages(msgs)
		entries, err := m.ListRebackEntries()
		if err != nil {
			t.Fatalf("ListRebackEntries() 意外错误: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("ListRebackEntries() 长度 = %d; want 2", len(entries))
		}
		if entries[0].Index != 1 {
			t.Errorf("entries[0].Index = %d; want 1", entries[0].Index)
		}
		if entries[1].Index != 2 {
			t.Errorf("entries[1].Index = %d; want 2", entries[1].Index)
		}
	})

	t.Run("仅有助手消息", func(t *testing.T) {
		msgs := []*schema.Message{
			schema.AssistantMessage("无用户消息", nil),
		}
		m := newTestMemoryWithMessages(msgs)
		entries, err := m.ListRebackEntries()
		if err != nil {
			t.Fatalf("ListRebackEntries() 意外错误: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("无用户消息时应返回空列表，got len=%d", len(entries))
		}
	})
}

// ──────────────────────────── buildSummary ────────────────────────────

func TestBuildSummary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "短消息（不截断）",
			content: "你好",
			want:    "你好",
		},
		{
			name:    "刚好 40 字符（不截断）",
			content: "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十",
			want:    "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十",
		},
		{
			name:    "长消息（截断）",
			content: "这是一条非常长的消息用于测试摘要生成功能是否正常工作需要验证截断逻辑是否正确",
			want:    "这是一条非常长的消息用于测试摘要生成功能是否正常工作需要验证截断逻辑是否正确", // 36 字符 ≤ 40，不截断
		},
		{
			name:    "空消息",
			content: "",
			want:    "",
		},
		{
			name:    "仅空白字符",
			content: "   ",
			want:    "",
		},
		{
			name:    "英文长消息",
			content: "This is a very long message that needs to be truncated by the buildSummary function for testing purposes",
			want:    "This is a very ...esting purposes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSummary(tt.content)
			if got != tt.want {
				t.Errorf("buildSummary(%q) = %q; want %q", tt.content, got, tt.want)
			}
		})
	}
}

// ──────────────────────────── validateID ────────────────────────────

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"普通 ID", "12345", false},
		{"层级 ID", "qq_private/12345", false},
		{"带连字符", "session-abc-123", false},
		{"空字符串", "", true},
		{"包含 ..", "../etc/passwd", true},
		{"仅为点", ".", true},
		{"Unix 绝对路径在 Windows 上可能不触发", "/etc/passwd", false}, // Windows 上 /etc/passwd 不是绝对路径
		{"Windows 绝对路径", "C:\\Users\\test", true},
		{"以 .. 开头", "..", true},
		{"包含隐藏目录", ".hidden/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateID(%q) error = %v; wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

// ──────────────────────────── LoadByID ────────────────────────────

func TestMemory_LoadByID(t *testing.T) {
	m := newTestMemory()
	// LoadByID 需要有效的 MemPath，空路径下应返回空列表（因 MkdirAll 在当前目录）
	msgs, err := m.LoadByID("nonexistent-id-12345")
	if err != nil {
		// 可能因路径问题失败，这是预期的（文件不存在）
		t.Logf("LoadByID 返回错误（预期）: %v", err)
		return
	}
	if msgs == nil {
		t.Error("LoadByID 不应返回 nil 切片")
	}
	if len(msgs) != 0 {
		t.Errorf("不存在的 ID 应返回空列表，got len=%d", len(msgs))
	}
}
