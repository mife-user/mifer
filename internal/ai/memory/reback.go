package memory

import (
	"encoding/json"
	"fmt"
	"mifer/pkg/errorer"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// RebackEntry 单个可回退的对话轮次摘要
type RebackEntry struct {
	Index   int
	Summary string
}

// ListRebackEntries 列出当前会话中所有用户消息，供回退选择使用
func (m *Memory) ListRebackEntries() ([]RebackEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var entries []RebackEntry
	userIdx := 0
	for _, msg := range m.Messages {
		if msg.Role == schema.User {
			userIdx++
			entries = append(entries, RebackEntry{
				Index:   userIdx,
				Summary: buildSummary(msg.Content),
			})
		}
	}
	return entries, nil
}

// Reback 回退到指定轮次之前：删除第 userMsgIndex 条用户消息及之后全部内容
//
// userMsgIndex 为 1-based 的用户消息轮次索引。
// 返回：被删除用户消息的摘要、完整内容。
// 内存中保留第 userMsgIndex 条用户消息之前的所有消息，JSONL 文件同步重写。
func (m *Memory) Reback(userMsgIndex int) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 定位第 userMsgIndex 个用户消息（1-based）
	userCount := 0
	targetIdx := -1
	for i, msg := range m.Messages {
		if msg.Role == schema.User {
			userCount++
			if userCount == userMsgIndex {
				targetIdx = i
				break
			}
		}
	}
	if targetIdx == -1 {
		if userMsgIndex <= 0 {
			return "", "", errorer.NewF(errorer.ErrIndexOutOfRange, fmt.Sprintf("无效的轮次: %d（最小为1）", userMsgIndex))
		}
		return "", "", errorer.NewF(errorer.ErrIndexOutOfRange, fmt.Sprintf("超过最大轮次: %d（共%d轮对话）", userMsgIndex, userCount))
	}

	summary := buildSummary(m.Messages[targetIdx].Content)
	fullContent := m.Messages[targetIdx].Content

	// 截断到第N条用户消息之前（不保留该用户消息，避免重发时上下文重复）
	m.Messages = m.Messages[:targetIdx]

	// 覆盖重写 JSONL 文件
	fileName := filepath.Join(m.Cfg.MemPath, fmt.Sprintf("%s.jsonl", m.Cfg.Id))
	f, err := os.Create(fileName)
	if err != nil {
		return "", "", errorer.NewS(errorer.ErrOpenFileFailed, err)
	}
	defer f.Close()

	for _, msg := range m.Messages {
		line, err := json.Marshal(msg)
		if err != nil {
			return "", "", errorer.NewS(errorer.ErrSerializeJSONFailed, err)
		}
		if _, err := f.Write(line); err != nil {
			return "", "", errorer.NewS(errorer.ErrWriteFileFailed, err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			return "", "", errorer.NewS(errorer.ErrWriteNewlineFailed, err)
		}
	}
	m.savedCount = len(m.Messages)
	return summary, fullContent, nil
}

// buildSummary 生成用户消息的简要摘要：前15字+后15字，短消息用全文
func buildSummary(content string) string {
	runes := []rune(content)
	if len(runes) <= 40 {
		return strings.TrimSpace(content)
	}
	head := 15
	tail := 15
	if len(runes) < head+tail {
		head = len(runes) / 2
		tail = len(runes) - head
	}
	return string(runes[:head]) + "..." + string(runes[len(runes)-tail:])
}
