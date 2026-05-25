package adminresp

// BackendStatus 单个后端模型的重载状态
type BackendStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`           // "ok" | "failed"
	Model  string `json:"model,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ReloadResp 配置重载响应
type ReloadResp struct {
	Success  bool            `json:"success"`
	Message  string          `json:"message"`
	Backends []BackendStatus `json:"backends"`
}
