package skill

import (
	"mifer/pkg/conf"

	"github.com/cloudwego/eino/adk"
)

// AgentHub 管理已注册的子 Agent 实例，供 fork 模式技能查找
type AgentHub struct {
	agents map[string]adk.Agent
}

// Skill 表示一个已加载的技能
type Skill struct {
	Name        string // 技能名（目录名）
	Description string // 技能描述（来自 frontmatter）
	Context     string // 执行模式: inline / fork
	Agent       string // fork 模式的目标 Agent 名
	Content     string // SKILL.md 中 frontmatter 之后的 markdown 内容
	BaseDir     string // 技能所在目录的绝对路径
}

// SkillInfo 技能简要信息（用于列表展示）
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Context     string `json:"context"`
	Agent       string `json:"agent"`
}

// Manager 技能管理器
type Manager struct {
	cfg       conf.SkillConfig
	skillsDir string
	skills    map[string]*Skill // name → Skill
}

// SkillTool 将技能系统适配为 Eino 的 tool.InvokableTool
type SkillTool struct {
	manager  *Manager
	agentHub *AgentHub
}
