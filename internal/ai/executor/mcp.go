package executor

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/logger"
)

// MCPStatus 返回所有 MCP Server 的连接状态
func (e *Executor) MCPStatus(ctx context.Context) (*domain.MCPStatusResp, error) {
	statuses := e.Humen.MCPManager.ListServers()

	var servers []domain.MCPServerStatus
	for _, s := range statuses {
		servers = append(servers, domain.MCPServerStatus{
			Name:      s.Name,
			Status:    s.Status,
			ToolCount: s.ToolCount,
			Error:     s.Error,
		})
	}

	if servers == nil {
		servers = []domain.MCPServerStatus{}
	}

	logger.Debug(ctx, "MCP状态查询完成", logger.S("count", string(rune(len(servers)+'0'))))
	return &domain.MCPStatusResp{Servers: servers}, nil
}
