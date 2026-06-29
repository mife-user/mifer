package adminresp

// StatusResp 后端状态查询响应，供 CLI/TUI 启动时检查模型就绪状态
type StatusResp struct {
	Ready    bool            `json:"ready"`              // default 后端是否可用
	Backends []BackendStatus `json:"backends"`           // 各后端加载状态
	Warnings []string        `json:"warnings,omitempty"` // 警告信息（如 api_key 未配置）
}
