package agentres

// PlanListRes 计划文件列表响应
type PlanListRes struct {
	Files []string `json:"files"`
}

// PlanLoadRes 计划文件内容响应
type PlanLoadRes struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
