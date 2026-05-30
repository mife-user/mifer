package skill

import (
	"fmt"

	"github.com/cloudwego/eino/adk"
)

// AgentHub 管理已注册的子 Agent 实例，供 fork 模式技能查找
type AgentHub struct {
	agents map[string]adk.Agent
}

// NewAgentHub 创建 AgentHub 实例
func NewAgentHub() *AgentHub {
	return &AgentHub{
		agents: make(map[string]adk.Agent),
	}
}

// Register 注册 Agent 实例
func (h *AgentHub) Register(name string, agent adk.Agent) {
	h.agents[name] = agent
}

// Get 按名称获取 Agent 实例
func (h *AgentHub) Get(name string) (adk.Agent, error) {
	agent, ok := h.agents[name]
	if !ok {
		return nil, fmt.Errorf("Agent [%s] 不存在，可用: %v", name, h.Names())
	}
	return agent, nil
}

// Names 返回所有已注册的 Agent 名称
func (h *AgentHub) Names() []string {
	names := make([]string, 0, len(h.agents))
	for name := range h.agents {
		names = append(names, name)
	}
	return names
}

// Has 检查 Agent 是否已注册
func (h *AgentHub) Has(name string) bool {
	_, ok := h.agents[name]
	return ok
}
