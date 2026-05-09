package domain

import (
	"encoding/json"
	"testing"
)

func TestTalkReq_JSON(t *testing.T) {
	req := TalkReq{Content: "你好"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var decoded TalkReq
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if decoded.Content != "你好" {
		t.Fatalf("期望 你好, 实际 %q", decoded.Content)
	}
}

func TestMemoryReq(t *testing.T) {
	req := MemoryReq{ID: 42}
	if req.ID != 42 {
		t.Fatalf("ID 应为 42, 实际 %d", req.ID)
	}
}
