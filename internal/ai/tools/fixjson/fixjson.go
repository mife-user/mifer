// Package fixjson 提供工具调用参数 JSON 的预处理修复。
// 当 LLM 生成的 JSON 参数存在常见格式问题（如字符串值中未转义的双引号）时，
// 尝试修复使其可通过 json.Unmarshal 解析，避免工具调用因参数格式错误而失败。
package fixjson

import (
	"context"
	"encoding/json"
	"strings"
)

// Handler 返回符合 compose.ToolArgumentsHandler 签名的预处理函数。
// 用于在 ToolsNodeConfig.ToolArgumentsHandler 中注册。
func Handler() func(ctx context.Context, name, arguments string) (string, error) {
	return func(_ context.Context, _ string, arguments string) (string, error) {
		return fixToolArguments(arguments), nil
	}
}

// fixToolArguments 尝试修复工具调用参数中的 JSON 格式问题。
// 若参数已是合法 JSON 则原样返回；
// 否则尝试修复常见问题（如字符串值中的未转义 ASCII 双引号），
// 修复后仍不合法则返回原始参数让上层报错。
func fixToolArguments(raw string) string {
	if json.Valid([]byte(raw)) {
		return raw
	}

	fixed := fixUnescapedQuotes(raw)
	if json.Valid([]byte(fixed)) {
		return fixed
	}

	// 修复失败，返回原始参数
	return raw
}

// fixUnescapedQuotes 修复 JSON 字符串值中未转义的 ASCII 双引号。
//
// 状态机：
//   - 进入字符串：读到 " 且不在字符串内 → 标记 inString=true
//   - 退出字符串：读到 " 且后跟结构字符（: , } ] 或 EOF/空白+结构字符）→ inString=false
//   - 内容引号：读到 " 但后跟非结构字符 → 转义为 \"
func fixUnescapedQuotes(raw string) string {
	var b strings.Builder
	b.Grow(len(raw) + 32)

	inString := false
	escape := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if escape {
			escape = false
			b.WriteByte(ch)
			continue
		}

		if ch == '\\' && inString {
			escape = true
			b.WriteByte(ch)
			continue
		}

		if ch == '"' {
			if !inString {
				// 进入字符串
				inString = true
				b.WriteByte(ch)
				continue
			}

			// 潜在的结构结束引号：检查后续字符
			next := nextStructural(raw, i+1)
			if next == 0 || next == ',' || next == ':' || next == '}' || next == ']' {
				// 后跟结构字符 → 真正的结束引号
				inString = false
				b.WriteByte(ch)
				continue
			}

			// 后跟非结构字符 → 内容中的引号，需要转义
			b.WriteString(`\"`)
			continue
		}

		b.WriteByte(ch)
	}

	return b.String()
}

// nextStructural 跳过空白，返回下一个非空白字符；到达末尾返回 0。
func nextStructural(raw string, start int) byte {
	for i := start; i < len(raw); i++ {
		ch := raw[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			continue
		}
		return ch
	}
	return 0
}
