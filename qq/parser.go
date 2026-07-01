package qq

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reCQAt    = regexp.MustCompile(`\[CQ:at,qq=\d+[^\]]*\]`)
	reCQImage = regexp.MustCompile(`\[CQ:image,[^\]]*\]`)
	reCQFace  = regexp.MustCompile(`\[CQ:face,[^\]]*\]`)
	reCQAny   = regexp.MustCompile(`\[CQ:[^\]]+\]`)
)

// cleanCQ 将 CQ 码清洗为 Agent 可理解的文本。
// [CQ:at,...] → 移除（At 检测由 isAtBot 独立处理）
// [CQ:image]  → "[图片]"
// [CQ:face]   → "[表情]"
// 其他 CQ 码  → 移除
func cleanCQ(raw string) string {
	s := reCQAt.ReplaceAllString(raw, "")
	s = reCQImage.ReplaceAllString(s, "[图片]")
	s = reCQFace.ReplaceAllString(s, "[表情]")
	s = reCQAny.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// isAtBot 检查消息中是否 @ 了指定 QQ 号。
func isAtBot(message string, botQQ int64) bool {
	return strings.Contains(message, fmt.Sprintf("[CQ:at,qq=%d]", botQQ))
}
