package client

import (
	"net/http"

	"mifer/cli/client/chathandler"
	"mifer/cli/client/clearhandler"
	"mifer/cli/client/excmemhandler"
	"mifer/cli/client/mcphandler"
	"mifer/cli/client/memhandler"
	"mifer/cli/client/planhandler"
	"mifer/cli/client/prompthandler"
	"mifer/cli/client/rebackhandler"
	"mifer/cli/client/reloadhandler"
	"mifer/cli/client/skillhandler"
)

const (
	APIChatPath   = "/api/ai/chat"
	APIMemoryPath = "/api/memory"
	APIExcmemPath = "/api/memory/exchange"
	APIClearPath  = "/api/memory/clear"
	APIPromptPath = "/api/prompt"
	APIRebackPath = "/api/memory/reback"
	APIReloadPath = "/api/admin/reload"
	APIPlanPath   = "/api/plan"
	APIMCPPath    = "/api/mcp"
	APISkillPath  = "/api/skill"
)

// Client HTTP API客户端，按服务聚合各Handler
type Client struct {
	Chat   *chathandler.ChatHandler
	Memory *memhandler.MemHandler
	Excmem *excmemhandler.ExcmemHandler
	Clear  *clearhandler.ClearHandler
	Reback *rebackhandler.RebackHandler
	Prompt *prompthandler.PromptHandler
	Reload *reloadhandler.ReloadHandler
	Plan   *planhandler.PlanHandler
	MCP    *mcphandler.MCPHandler
	Skill  *skillhandler.SkillHandler
}

// New 创建API客户端实例
func New(baseURL string) *Client {
	httpClient := http.DefaultClient
	return &Client{
		Chat:   chathandler.New(httpClient, baseURL+APIChatPath),
		Memory: memhandler.New(httpClient, baseURL+APIMemoryPath),
		Excmem: excmemhandler.New(httpClient, baseURL+APIExcmemPath),
		Clear:  clearhandler.New(httpClient, baseURL+APIClearPath),
		Reback: rebackhandler.New(httpClient, baseURL+APIRebackPath),
		Prompt: prompthandler.New(httpClient, baseURL+APIPromptPath),
		Reload: reloadhandler.New(httpClient, baseURL+APIReloadPath),
		Plan:   planhandler.New(httpClient, baseURL+APIPlanPath),
		MCP:    mcphandler.New(httpClient, baseURL+APIMCPPath+"/status"),
		Skill:  skillhandler.New(httpClient, baseURL+APISkillPath+"/list"),
	}
}
