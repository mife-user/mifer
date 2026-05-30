package mcp

import (
	"context"
	"fmt"
	"sync"

	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerInstance 表示单个 MCP Server 的连接实例
type ServerInstance struct {
	Config conf.MCPServerConfig
	Client client.MCPClient
	Tools  []tool.InvokableTool
	Status string // connected / disabled / error / disconnected
	ErrMsg string
	cancel context.CancelFunc
}

// Manager 管理所有 MCP Server 连接的生命周期
type Manager struct {
	mu      sync.RWMutex
	servers map[string]*ServerInstance // name → 实例
}

// NewManager 根据配置创建 MCP 连接管理器并启动所有已启用的 Server
func NewManager(cfgs []conf.MCPServerConfig) *Manager {
	m := &Manager{
		servers: make(map[string]*ServerInstance),
	}
	for _, cfg := range cfgs {
		if cfg.Enabled {
			m.startServer(cfg)
		} else {
			m.servers[cfg.Name] = &ServerInstance{
				Config: cfg,
				Status: StatusDisabled,
			}
		}
	}
	return m
}

// startServer 启动单个 MCP Server（内部方法，不加锁）
func (m *Manager) startServer(cfg conf.MCPServerConfig) {
	inst := &ServerInstance{
		Config: cfg,
		Status: StatusConnected,
	}

	// 创建 stdio MCP 客户端
	ctx, cancel := context.WithCancel(context.Background())
	inst.cancel = cancel

	cli, err := client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
	if err != nil {
		inst.Status = StatusError
		inst.ErrMsg = "创建客户端失败: " + err.Error()
		logger.Error("MCP Server 启动失败: "+cfg.Name, logger.C(err))
		m.servers[cfg.Name] = inst
		return
	}
	inst.Client = cli

	// 握手初始化
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "Mifer",
		Version: "1.0.0",
	}

	_, err = cli.Initialize(ctx, initReq)
	if err != nil {
		inst.Status = StatusError
		inst.ErrMsg = "握手失败: " + err.Error()
		logger.Error("MCP Server 握手失败: "+cfg.Name, logger.C(err))
		m.servers[cfg.Name] = inst
		return
	}

	// 获取工具列表
	toolsResp, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		inst.Status = StatusError
		inst.ErrMsg = "获取工具列表失败: " + err.Error()
		logger.Error("MCP Server 获取工具列表失败: "+cfg.Name, logger.C(err))
		m.servers[cfg.Name] = inst
		return
	}

	// 适配所有工具
	for _, t := range toolsResp.Tools {
		adapter := NewMCPToolAdapter(t, cli, cfg.Name)
		inst.Tools = append(inst.Tools, adapter)
	}

	logger.Info(fmt.Sprintf("MCP Server [%s] 连接成功，加载 %d 个工具", cfg.Name, len(inst.Tools)))
	m.servers[cfg.Name] = inst
}

// stopServer 停止单个 MCP Server（内部方法，不加锁）
func (m *Manager) stopServer(name string) {
	inst, ok := m.servers[name]
	if !ok {
		return
	}
	if inst.cancel != nil {
		inst.cancel()
	}
	if inst.Client != nil {
		inst.Client.Close()
	}
	inst.Status = StatusDisconnected
	logger.Info("MCP Server 已断开: " + name)
}

// Reload 热加载 MCP 配置，对比新旧配置增/删/重启 Server
func (m *Manager) Reload(cfgs []conf.MCPServerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newCfgMap := make(map[string]conf.MCPServerConfig)
	for _, cfg := range cfgs {
		newCfgMap[cfg.Name] = cfg
	}

	// 删除旧配置中不存在的 Server
	for name := range m.servers {
		if _, ok := newCfgMap[name]; !ok {
			m.stopServer(name)
			delete(m.servers, name)
		}
	}

	// 处理新增和配置变更的 Server
	for name, cfg := range newCfgMap {
		old, exists := m.servers[name]
		if !exists {
			// 新增
			if cfg.Enabled {
				m.startServer(cfg)
			} else {
				m.servers[name] = &ServerInstance{
					Config: cfg,
					Status: StatusDisabled,
				}
			}
		} else if cfgChanged(old, cfg) {
			// 配置变更：停旧启新
			m.stopServer(name)
			if cfg.Enabled {
				m.startServer(cfg)
			} else {
				m.servers[name] = &ServerInstance{
					Config: cfg,
					Status: StatusDisabled,
				}
			}
		}
		// 配置未变，跳过
	}
}

// cfgChanged 检查 Server 配置是否发生变更
func cfgChanged(old *ServerInstance, newCfg conf.MCPServerConfig) bool {
	if old.Config.Command != newCfg.Command {
		return true
	}
	if old.Config.Enabled != newCfg.Enabled {
		return true
	}
	if !stringSliceEqual(old.Config.Args, newCfg.Args) {
		return true
	}
	if !stringSliceEqual(old.Config.Env, newCfg.Env) {
		return true
	}
	if !stringSliceEqual(old.Config.Agents, newCfg.Agents) {
		return true
	}
	return false
}

// stringSliceEqual 比较两个字符串切片是否相等
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GetToolsForAgent 获取分配给指定 Agent 的所有 MCP 工具
func (m *Manager) GetToolsForAgent(agentName string) []tool.InvokableTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []tool.InvokableTool
	for _, inst := range m.servers {
		if inst.Status != StatusConnected {
			continue
		}
		// 检查是否分配给该 Agent
		if !isAgentTargeted(inst.Config.Agents, agentName) {
			continue
		}
		tools = append(tools, inst.Tools...)
	}
	return tools
}

// isAgentTargeted 判断 Agent 是否在分配列表中（空或 ["*"] 表示全部）
func isAgentTargeted(agents []string, agentName string) bool {
	if len(agents) == 0 {
		return true
	}
	for _, a := range agents {
		if a == "*" || a == agentName {
			return true
		}
	}
	return false
}

// ListServers 返回所有 MCP Server 的状态
func (m *Manager) ListServers() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var statuses []ServerStatus
	for _, inst := range m.servers {
		s := ServerStatus{
			Name:      inst.Config.Name,
			Status:    inst.Status,
			ToolCount: len(inst.Tools),
			Error:     inst.ErrMsg,
		}
		statuses = append(statuses, s)
	}
	return statuses
}

// Close 关闭所有 MCP Server 连接
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name := range m.servers {
		m.stopServer(name)
	}
}
