package domain

import "context"

type AgentService interface {
	Chat(ctx context.Context, req *Agent) (*Agent, error)
}
