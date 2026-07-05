package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

// CountToolMessages 统计记忆中的工具消息总数（含 assistant+ToolCalls 与 tool 结果）。
func (m *Memory) CountToolMessages() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return countToolMessages(m.messages)
}

// countToolMessages 内部无锁版本，供 compressToolHistory 复用。
func countToolMessages(messages []*schema.Message) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == schema.Tool || (msg.Role == schema.Assistant && len(msg.ToolCalls) > 0) {
			count++
		}
	}
	return count
}

// CompressToolHistory 压缩旧轮次中的工具调用消息。
//
// 保留最近 keepRounds 个用户轮次内的工具调用完整内容，
// 更早的工具调用块替换为一条摘要 assistant 消息。
// 调用方需持有 chatMu 保护并发安全。
//
// 返回 true 表示实际执行了压缩（调用方无需再调用 Save）。
func (m *Memory) CompressToolHistory(keepRounds int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if keepRounds <= 0 {
		return false
	}

	// 以用户消息为分界，找到最近 keepRounds 轮的起始位置
	userCount := 0
	splitIdx := 0
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == schema.User {
			userCount++
			if userCount >= keepRounds {
				splitIdx = i
				break
			}
		}
	}

	if splitIdx == 0 {
		return false // 没有足够多的轮次需要压缩
	}

	oldMsgs := m.messages[:splitIdx]
	recentMsgs := m.messages[splitIdx:]

	// 统计旧消息中的工具调用
	toolCount := countToolMessages(oldMsgs)
	if toolCount == 0 {
		return false
	}

	// 分析旧轮次中的工具调用，生成摘要
	toolSummary := buildToolSummary(oldMsgs)
	if toolSummary == "" {
		return false
	}

	// 构建新消息列表：保留旧轮次中的对话（user + 纯文本 assistant），
	// 移除工具消息，插入工具摘要
	var compressed []*schema.Message
	for _, msg := range oldMsgs {
		if msg.Role == schema.Tool {
			continue
		}
		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			continue
		}
		compressed = append(compressed, msg)
	}
	compressed = append(compressed, schema.AssistantMessage(toolSummary, nil))
	compressed = append(compressed, recentMsgs...)

	m.messages = compressed

	// 重写持久化文件（与 ReplaceMessages 逻辑一致）
	fileName, err := buildFilePath(m.Cfg.MemPath, m.Cfg.Id)
	if err != nil {
		return false
	}
	if err := rewriteFile(fileName, m.messages); err != nil {
		return false
	}
	m.savedCount = len(m.messages)

	return true
}

// rewriteFile 将消息列表全量写入 JSONL 文件（先写临时文件再 Rename 保证原子性）。
// 与 ReplaceMessages / Reback 使用相同的持久化格式。
func rewriteFile(fileName string, messages []*schema.Message) error {
	tmpPath := fileName + ".tmp"
	defer os.Remove(tmpPath)

	f, err := os.Create(tmpPath)
	if err != nil {
		logger.Error("创建临时记忆文件失败", logger.S("path", tmpPath), logger.C(err))
		return errorer.NewS(errorer.ErrOpenFileFailed, err)
	}

	writeErr := func() error {
		defer f.Close()
		for _, msg := range messages {
			line, err := json.Marshal(msg)
			if err != nil {
				return errorer.NewS(errorer.ErrSerializeJSONFailed, err)
			}
			if _, err := f.Write(line); err != nil {
				return errorer.NewS(errorer.ErrWriteFileFailed, err)
			}
			if _, err := f.Write([]byte("\n")); err != nil {
				return errorer.NewS(errorer.ErrWriteNewlineFailed, err)
			}
		}
		return nil
	}()
	if writeErr != nil {
		return writeErr
	}

	if err := os.Rename(tmpPath, fileName); err != nil {
		logger.Error("重命名临时记忆文件失败", logger.S("path", fileName), logger.C(err))
		return errorer.NewS(errorer.ErrFileRenameFailed, err)
	}

	return nil
}

// buildToolSummary 分析消息列表中的工具调用，生成统计摘要。
// 格式："[工具调用摘要] 早期对话中使用了以下工具: knowledge_search(3次), file_reader(1次)"
func buildToolSummary(messages []*schema.Message) string {
	toolCounts := make(map[string]int)
	for _, msg := range messages {
		if msg.Role == schema.Tool && msg.ToolName != "" {
			toolCounts[msg.ToolName]++
		} else if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				toolCounts[tc.Function.Name]++
			}
		}
	}

	if len(toolCounts) == 0 {
		return ""
	}

	var parts []string
	for name, count := range toolCounts {
		parts = append(parts, fmt.Sprintf("%s(%d次)", name, count))
	}
	return "[工具调用摘要] 早期对话中使用了以下工具: " + strings.Join(parts, "、")
}
