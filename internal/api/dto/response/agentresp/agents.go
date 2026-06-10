package agentres

// AgentInfo agent 基础信息
type AgentInfo struct {
	Name         string   `json:"name"`
	ModelBackend string   `json:"model_backend"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Description  string   `json:"description"`
	Tools        []string `json:"tools"`
}

// AgentListRes agent 列表响应
type AgentListRes struct {
	Agents []AgentInfo `json:"agents"`
}
