package agentservice

import (
	"mifer/internal/ai/executor"
	"mifer/internal/domain"
)

type AgentService struct {
	Executor domain.Agent
}

func NewAgentService(exec domain.Agent) domain.AgentService {
	return &AgentService{Executor: exec}
}

// CloseExecutor 释放底层 Executor 持有的资源（MCP 子进程、确认存储 actor 等）。
// 调用后此 AgentService 不应再被使用。供 reload 等场景清理旧实例使用。
func (s *AgentService) CloseExecutor() {
	if exec, ok := s.Executor.(*executor.Executor); ok {
		exec.Close()
	}
}
