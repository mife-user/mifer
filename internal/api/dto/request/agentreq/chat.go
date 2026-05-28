package agentreq

type ChatReq struct {
	Content  string `json:"content"`
	PlanMode bool   `json:"plan_mode,omitempty"`
}
