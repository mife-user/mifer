package agentres

// SkillInfo 技能简要信息
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Context     string `json:"context"`
	Agent       string `json:"agent"`
}

// SkillListRes 技能列表响应
type SkillListRes struct {
	Skills []SkillInfo `json:"skills"`
}
