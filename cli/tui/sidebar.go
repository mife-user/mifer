package tui

import (
	"fmt"
	"time"

	"mifer/pkg/logger"
)

// SidebarState 右侧状态侧边栏状态，跟踪 agent/工具执行情况与 token 消耗
type SidebarState struct {
	Current string          // 当前活跃项（agent名 或 "  tool名"），空=无
	Log     []string        // 状态事件日志（带时间戳，最新追加）
	Token   *TokenUsageData // 最新 token 统计（nil=无数据）
}

// update 根据流式状态消息更新侧边栏状态
func (s *SidebarState) update(msg streamStatusMsg, showTiming bool, maxLog int) {
	switch msg.event {
	case "agent_start":
		logger.Info("agent_start: %s", logger.S("agentName", msg.name))
		if s.Current != "" {
			s.appendLog(showTiming, "%s 完成", s.Current)
		}
		s.Current = msg.name
		s.appendLog(showTiming, "%s 开始", msg.name)
	case "agent_end":
		logger.Info("agent_end: %s", logger.S("agentName", msg.name))
		if s.Current == msg.name {
			s.appendLog(showTiming, "%s 完成", s.Current)
			s.Current = ""
		}
	case "tool_start":
		logger.Info("tool_start: %s", logger.S("toolName", msg.name))
		if s.Current != "" && s.Current != "  "+msg.name {
			// 工具切换：先结束上一个工具
		}
		s.Current = "  " + msg.name
		s.appendLog(showTiming, "  %s 开始", msg.name)
	case "tool_end":
		logger.Info("tool_end: %s", logger.S("toolName", msg.name))
		suffix := ""
		if msg.errMsg != "" {
			suffix = " [ERROR]"
		}
		s.appendLog(showTiming, "  %s 完成%s", msg.name, suffix)
		if s.Current == "  "+msg.name {
			s.Current = ""
		}
	case "tool_error":
		logger.Info("tool_error: %s", logger.S("toolErr", msg.errMsg))
		// 错误已在 tool_end 中标记，此处记录详情
		if msg.errMsg != "" {
			s.appendLog(showTiming, "  E: %s", msg.errMsg)
		}
	case "token":
		if msg.tokenUsage != nil {
			s.Token = msg.tokenUsage
		}
	}
	// 裁剪日志到最大行数
	if maxLog > 0 && len(s.Log) > maxLog {
		s.Log = s.Log[len(s.Log)-maxLog:]
	}
}

func (s *SidebarState) appendLog(showTiming bool, format string, args ...interface{}) {
	entry := fmt.Sprintf(format, args...)
	if showTiming {
		entry = time.Now().Format("15:04:05") + " " + entry
	}
	s.Log = append(s.Log, entry)
}
