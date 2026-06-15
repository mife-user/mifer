package skill

import (
	"os"
	"path/filepath"
)

// builtinContextSummarizer 内置技能：对话历史压缩总结的 SKILL.md 完整内容
const builtinContextSummarizer = `---
name: context-summarizer
description: 内置技能：对话历史压缩总结，提取关键信息、决策和待办事项
context: inline
---

# 上下文压缩

你是一个专业的对话总结助手。请将以下对话历史压缩为简洁但信息完整的摘要。

## 总结要求

1. **保留关键信息**：
   - 用户的核心需求与目标
   - 已完成的任务和操作（文件创建、代码修改、配置变更等）
   - 重要的技术决策和理由
   - 当前存在的未解决问题或待办事项

2. **删除冗余内容**：
   - 重复的确认和问候
   - 中间过程的工具调用细节（保留最终结果）
   - 用户取消或已放弃的请求

3. **输出格式**：
   - 使用 markdown 列表格式
   - 按主题分节（目标、已完成、进行中、决策记录）
   - 总长度不超过 500 字

4. **语言**：使用中文总结，技术术语保留英文原文`

// copyBuiltinSkills 将内置技能从常量写入技能目录
func (m *Manager) copyBuiltinSkills() error {
	dir := filepath.Join(m.skillsDir, "context-summarizer")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	skillFile := filepath.Join(dir, "SKILL.md")
	return os.WriteFile(skillFile, []byte(builtinContextSummarizer), 0644)
}
