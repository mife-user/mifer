package prompt

import (
	"context"
	"testing"

	"mifer/internal/ai/memory"

	"github.com/cloudwego/eino/schema"
)

// ============================================================================
// prompt_test.go — prompt 包单元测试
// 测试纯函数部分：buildLegacy、getter/setter、Build（不含文件 I/O）
// ============================================================================

// newTestMemory 创建用于测试的空 Memory（使用临时路径避免文件 I/O）
func newTestMemory() *memory.Memory {
	m, _ := memory.Init("test-session-for-prompt-test")
	return m
}

// newTestPrompty 创建一个不含 Template 的 Prompty 用于测试 buildLegacy 路径
func newTestPrompty(sysPrompt string) *Prompty {
	return &Prompty{
		Memory: &memory.Memory{},
		// Template 为 nil，Build 将走 buildLegacy 回退路径
		SystemPrompt: sysPrompt,
	}
}

// ──────────────────────────── GetSystemPrompt / SetSystemPrompt ────────────────────────────

func TestPrompty_GetSystemPrompt(t *testing.T) {
	p := &Prompty{SystemPrompt: "测试提示词"}
	got := p.GetSystemPrompt()
	if got != "测试提示词" {
		t.Errorf("GetSystemPrompt() = %q; want %q", got, "测试提示词")
	}
}

func TestPrompty_SetSystemPrompt(t *testing.T) {
	p := &Prompty{SystemPrompt: "旧提示词"}
	p.SetSystemPrompt("新提示词")
	if p.SystemPrompt != "新提示词" {
		t.Errorf("SetSystemPrompt 后 SystemPrompt = %q; want %q", p.SystemPrompt, "新提示词")
	}
}

func TestPrompty_SetSystemPrompt_Empty(t *testing.T) {
	p := &Prompty{SystemPrompt: "旧提示词"}
	p.SetSystemPrompt("")
	if p.SystemPrompt != "" {
		t.Errorf("SetSystemPrompt(\"\") 后 SystemPrompt = %q; want \"\"", p.SystemPrompt)
	}
}

// ──────────────────────────── buildLegacy ────────────────────────────

func TestPrompty_BuildLegacy(t *testing.T) {
	tests := []struct {
		name        string
		sysPrompt   string
		query       string
		wantMsgCnt  int           // 期望消息总数
		firstIsSys  bool          // 第一条是否为 System 消息
		lastIsUser  bool          // 最后一条是否为 User 消息
		lastContent string        // 最后一条消息内容
	}{
		{
			name:       "有系统提示词",
			sysPrompt:  "你是一个助手",
			query:      "你好",
			wantMsgCnt: 2,
			firstIsSys: true,
			lastIsUser: true,
		},
		{
			name:       "无系统提示词",
			sysPrompt:  "",
			query:      "你好",
			wantMsgCnt: 1,
			firstIsSys: false,
			lastIsUser: true,
		},
		{
			name:       "空查询",
			sysPrompt:  "系统提示",
			query:      "",
			wantMsgCnt: 2,
			firstIsSys: true,
			lastIsUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPrompty(tt.sysPrompt)
			msgs := p.buildLegacy(tt.query)

			if len(msgs) != tt.wantMsgCnt {
				t.Errorf("buildLegacy() 返回消息数 = %d; want %d", len(msgs), tt.wantMsgCnt)
			}
			if len(msgs) == 0 {
				return
			}

			if tt.firstIsSys && msgs[0].Role != schema.System {
				t.Errorf("第一条消息 Role = %s; want System", msgs[0].Role)
			}
			if tt.lastIsUser && msgs[len(msgs)-1].Role != schema.User {
				t.Errorf("最后一条消息 Role = %s; want User", msgs[len(msgs)-1].Role)
			}
			if msgs[len(msgs)-1].Content != tt.query {
				t.Errorf("最后一条消息 Content = %q; want %q", msgs[len(msgs)-1].Content, tt.query)
			}
		})
	}
}

func TestPrompty_BuildLegacy_WithMemory(t *testing.T) {
	// 模拟有对话历史的场景
	mem := &memory.Memory{}
	mem.AppendUser("上一轮问题")
	mem.AppendAssistant("上一轮回答")

	p := &Prompty{
		Memory:       mem,
		SystemPrompt: "系统提示",
		Template:     nil, // 走 buildLegacy 路径
	}

	msgs := p.buildLegacy("当前问题")

	// System + 2条历史 + 当前问题 = 4 条消息
	if len(msgs) != 4 {
		t.Fatalf("buildLegacy() 返回消息数 = %d; want 4", len(msgs))
	}
	if msgs[0].Role != schema.System {
		t.Errorf("第1条应为 System，got %s", msgs[0].Role)
	}
	if msgs[1].Role != schema.User || msgs[1].Content != "上一轮问题" {
		t.Errorf("第2条应为 User(上一轮问题)，got %s: %q", msgs[1].Role, msgs[1].Content)
	}
	if msgs[2].Role != schema.Assistant || msgs[2].Content != "上一轮回答" {
		t.Errorf("第3条应为 Assistant(上一轮回答)，got %s: %q", msgs[2].Role, msgs[2].Content)
	}
	if msgs[3].Role != schema.User || msgs[3].Content != "当前问题" {
		t.Errorf("第4条应为 User(当前问题)，got %s: %q", msgs[3].Role, msgs[3].Content)
	}
}

// ──────────────────────────── Build (走 template 路径) ────────────────────────────

func TestPrompty_Build_WithTemplate(t *testing.T) {
	// 构造带 Template 的 Prompty，测试 Build 的 template 分支
	mem := &memory.Memory{}
	mem.AppendUser("历史问题")
	mem.AppendAssistant("历史回答")

	tmpl := newDefaultTemplate()
	if tmpl == nil {
		t.Fatal("newDefaultTemplate() 返回 nil")
	}

	p := &Prompty{
		Memory:       mem,
		SystemPrompt: "系统提示词内容",
		Template:     tmpl,
	}

	ctx := context.Background()
	msgs, err := p.Build(ctx, "当前用户输入")
	if err != nil {
		t.Fatalf("Build() 意外错误: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("Build() 返回空消息列表")
	}

	// 验证消息结构：System → MessagesPlaceholder(历史) → User(当前)
	if msgs[0].Role != schema.System {
		t.Errorf("第1条应为 System，got %s", msgs[0].Role)
	}
	if msgs[0].Content != "系统提示词内容" {
		t.Errorf("System 内容 = %q; want %q", msgs[0].Content, "系统提示词内容")
	}
	// 最后一条应为用户消息
	last := msgs[len(msgs)-1]
	if last.Role != schema.User || last.Content != "当前用户输入" {
		t.Errorf("最后一条应为 User(当前用户输入)，got %s: %q", last.Role, last.Content)
	}
}

// ──────────────────────────── newDefaultTemplate ────────────────────────────

func TestNewDefaultTemplate(t *testing.T) {
	tmpl := newDefaultTemplate()
	if tmpl == nil {
		t.Fatal("newDefaultTemplate() 返回 nil")
	}

	// 验证模板可以正常格式化
	ctx := context.Background()
	msgs, err := tmpl.Format(ctx, map[string]any{
		"system_prompt": "测试系统提示",
		"history":       []*schema.Message{schema.UserMessage("历史消息")},
		"query":         "用户问题",
	})
	if err != nil {
		t.Fatalf("模板格式化失败: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("模板应生成 3 条消息，got %d", len(msgs))
	}
}
