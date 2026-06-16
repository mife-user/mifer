package client

import (
	"net/http"

	"mifer/cli/client/agentshandler"
	"mifer/cli/client/chathandler"
	"mifer/cli/client/clearhandler"
	"mifer/cli/client/compactorhandler"
	"mifer/cli/client/excmemhandler"
	"mifer/cli/client/mcphandler"
	"mifer/cli/client/memhandler"
	"mifer/cli/client/planhandler"
	"mifer/cli/client/prompthandler"
	"mifer/cli/client/questionhandler"
	"mifer/cli/client/rebackhandler"
	"mifer/cli/client/reloadhandler"
	"mifer/cli/client/skillhandler"
	"mifer/cli/client/toolconfirmhandler"
)

const (
	APIChatPath    = "/api/ai/chat"
	APIMemoryPath  = "/api/memory"
	APIExcmemPath  = "/api/memory/exchange"
	APIClearPath   = "/api/memory/clear"
	APICompactPath = "/api/memory/compact"
	APIPromptPath  = "/api/prompt"
	APIRebackPath  = "/api/memory/reback"
	APIReloadPath  = "/api/admin/reload"
	APIPlanPath    = "/api/plan"
	APIMCPPath     = "/api/mcp"
	APISkillPath   = "/api/skill"
	APIAgentsPath  = "/api/agents"
)

// Client HTTP API客户端，按服务聚合各Handler
type Client struct {
	Chat         *chathandler.ChatHandler
	Memory       *memhandler.MemHandler
	Excmem       *excmemhandler.ExcmemHandler
	Clear        *clearhandler.ClearHandler
	Compact      *compactorhandler.CompactHandler
	Reback       *rebackhandler.RebackHandler
	Prompt       *prompthandler.PromptHandler
	Reload       *reloadhandler.ReloadHandler
	Plan         *planhandler.PlanHandler
	MCP          *mcphandler.MCPHandler
	Skill        *skillhandler.SkillHandler
	Agents       *agentshandler.AgentsHandler
	ToolConfirm  *toolconfirmhandler.ConfirmHandler
	AllowlistAdd *toolconfirmhandler.AllowlistHandler
	Question     *questionhandler.QuestionHandler
}

// New 创建API客户端实例
func New(baseURL string) *Client {
	httpClient := http.DefaultClient
	return &Client{
		Chat:         chathandler.New(httpClient, baseURL+APIChatPath),
		Memory:       memhandler.New(httpClient, baseURL+APIMemoryPath),
		Excmem:       excmemhandler.New(httpClient, baseURL+APIExcmemPath),
		Clear:        clearhandler.New(httpClient, baseURL+APIClearPath),
		Compact:      compactorhandler.New(httpClient, baseURL+APICompactPath),
		Reback:       rebackhandler.New(httpClient, baseURL+APIRebackPath),
		Prompt:       prompthandler.New(httpClient, baseURL+APIPromptPath),
		Reload:       reloadhandler.New(httpClient, baseURL+APIReloadPath),
		Plan:         planhandler.New(httpClient, baseURL+APIPlanPath),
		MCP:          mcphandler.New(httpClient, baseURL+APIMCPPath+"/status"),
		Skill:        skillhandler.New(httpClient, baseURL+APISkillPath+"/list"),
		Agents:       agentshandler.New(httpClient, baseURL+APIAgentsPath),
		ToolConfirm:  toolconfirmhandler.New(httpClient, baseURL),
		AllowlistAdd: toolconfirmhandler.NewAllowlist(httpClient, baseURL),
		Question:     questionhandler.New(httpClient, baseURL),
	}
}
