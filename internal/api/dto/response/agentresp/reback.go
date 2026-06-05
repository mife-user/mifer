package agentres

// RebackEntryRes 可回退的对话轮次
type RebackEntryRes struct {
	Index   int    `json:"index"`
	Summary string `json:"summary"`
}

// RebackListRes 回退列表响应
type RebackListRes struct {
	Entries []RebackEntryRes `json:"entries"`
}

// RebackRes 回退结果响应
type RebackRes struct {
	Summary string `json:"summary"`
	Content string `json:"content"`
	Message string `json:"message"`
}
