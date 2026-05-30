# MCP 客户端功能设计

> 日期：2026-05-30 | 状态：待评审

## 一、目标

为 Mifer 添加 MCP（Model Context Protocol）客户端能力，使其能够连接外部 MCP Server（stdio 传输），将外部工具注入到现有 Agent 体系中，支持配置文件管理 + 运行时热加载。

## 二、架构概览

在现有分层架构上增加 `pkg/mcp/` 适配层，将 MCP 工具包装为 Eino `tool.InvokableTool`，按配置分配注入到各子 Agent。

```
                  ┌─────────────────────────────────┐
                  │        Mifer (Orchestrator)       │
                  │  deep.New → SubAgents             │
                  └──────────┬──────────────────────┘
                             │
        ┌────────────────────┼──────────────────────┐
        │                    │                       │
   MiEditer           MiCommander              MiAuditor ...
   (文件工具)          (命令工具)               (审计工具)
        │                    │                       │
   + [MCP工具A]       + [MCP工具B]             + [MCP工具C]
        │                    │                       │
        └────────────────────┼──────────────────────┘
                             │
                    ┌────────┴────────┐
                    │  pkg/mcp/       │
                    │  Manager        │
                    │  - Reload()     │
                    │  - GetTools()   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         Server A       Server B       Server C
         (stdio)        (stdio)         (stdio)
```

**设计原则：**
- 不改动 Agent 框架，MCP 工具像内置工具一样通过闭包注入
- 不破坏性改动现有代码，所有新增功能通过独立包和扩展点实现
- MCP Server 连接失败不阻塞启动和热加载，降级跳过

## 三、配置结构

### 类型定义（`pkg/conf/type.go` 追加）

```go
type MCPConfig struct {
    Servers []MCPServerConfig `mapstructure:"servers"`
}

type MCPServerConfig struct {
    Name    string   `mapstructure:"name"`    // 唯一标识，如 "filesystem"
    Command string   `mapstructure:"command"` // 启动命令，如 "npx"
    Args    []string `mapstructure:"args"`    // 参数，如 ["-y", "@anthropic/mcp-server-filesystem", "/path"]
    Env     []string `mapstructure:"env"`     // 环境变量（可选），如 ["GITHUB_TOKEN=xxx"]
    Agents  []string `mapstructure:"agents"`  // 分配给哪些 Agent，空或 ["*"] 表示全部
    Enabled bool     `mapstructure:"enabled"` // 是否启用，默认 true
}
```

在全局 `Config` 结构体中追加字段：`Mcp MCPConfig `mapstructure:"mcp"``

### 配置文件示例

```yaml
mcp:
  servers:
    - name: "filesystem"
      command: "npx"
      args: ["-y", "@anthropic/mcp-server-filesystem", "/home/user/projects"]
      agents: ["MiEditer", "MiAuditor"]
      enabled: true

    - name: "github"
      command: "npx"
      args: ["-y", "@anthropic/mcp-server-github"]
      env: ["GITHUB_TOKEN=ghp_xxx"]
      agents: ["MiPlanner"]
      enabled: false
```

### 默认值（`pkg/conf/new.go` defaultConfig 追加）

```yaml
mcp:
  servers: []
```

## 四、核心模块设计

### 4.1 Manager — MCP 连接管理器（`pkg/mcp/manager.go`）

负责所有 MCP Server 连接的生命周期管理。

```go
type Manager struct {
    mu      sync.RWMutex
    servers map[string]*ServerInstance  // name → 实例
}

// 核心方法
func NewManager(cfgs []MCPServerConfig) *Manager
func (m *Manager) Reload(cfgs []MCPServerConfig) error  // 热加载
func (m *Manager) GetToolsForAgent(agentName string) []tool.InvokableTool
func (m *Manager) ListServers() []ServerStatus
func (m *Manager) Close() error
```

**Reload 流程：**
1. 对比新旧配置，计算差异（新增/删除/变更）
2. 已删除的 Server → `StopServer()` → 从 map 移除
3. 配置变更的 Server → `StopServer()` + `StartServer()`
4. 新增的 Server → `StartServer()` → 加入 map
5. 不变的 Server → 跳过
6. 启动失败记录日志，不阻塞其他 Server

### 4.2 ServerInstance — 单个 Server 实例（`pkg/mcp/manager.go`）

```go
type ServerInstance struct {
    Config   MCPServerConfig
    Client   client.MCPClient         // mcp-go stdio 客户端
    Tools    []tool.InvokableTool     // 已适配的 Eino 工具
    Status   ServerStatus
    Cmd      *exec.Cmd               // 子进程句柄
    Cancel   context.CancelFunc
}

type ServerStatus struct {
    Name     string `json:"name"`
    Status   string `json:"status"`   // connected / disconnected / error
    ToolCount int   `json:"tool_count"`
    Error    string `json:"error,omitempty"`
}
```

**StartServer 流程：**
1. 通过 `client.NewStdioMCPClient(command, args, env...)` 创建 stdio 客户端
2. `client.Initialize()` 握手
3. `client.ListTools()` 获取工具列表
4. 遍历工具列表，每个通过 `MCPToolAdapter` 包装为 Eino `tool.InvokableTool`
5. 工具名加前缀 `{serverName}_{toolName}`，避免跨 Server 同名冲突

### 4.3 MCPToolAdapter — 工具适配器（`pkg/mcp/adapter.go`）

将 MCP `Tool` 适配为 Eino `tool.InvokableTool` 接口。

```go
type MCPToolAdapter struct {
    mcpTool    mcp.Tool
    mcpClient  client.MCPClient
    serverName string
    fullName   string  // {serverName}_{toolName}
}

// Info 返回 Eino ToolInfo，将 MCP JSON Schema 转换为 Eino ParameterInfo
func (a *MCPToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error)

// InvokableRun 执行工具：Deserialize argsJSON → CallToolRequest → CallTool → 提取文本结果
func (a *MCPToolAdapter) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error)
```

**关键处理：**
- **InputSchema 转换**：MCP 使用标准 JSON Schema（`{"type":"object","properties":{...},"required":[...]}`），需映射为 Eino 的 `schema.ParameterInfo` 结构（Type/SubParams/Required/Enum 等字段）
- **错误透传**：`CallToolResult.IsError=true` 时，将错误内容作为正常字符串返回（而非 Go error），让 LLM 可以看到工具的错误信息并自行修正。MCP 协议规定工具错误应放在 result 而非 protocol error
- **Content 提取**：从 `CallToolResult.Content` 数组中提取 `TextContent` 文本，多个文本块拼接返回

### 4.4 工具名命名规则

- 格式：`{serverName}_{toolName}`
- 示例：`filesystem` server 的 `read_file` 工具 → `filesystem_read_file`
- 目的：避免不同 MCP Server 的同名工具冲突，同时也让 LLM 了解工具来源

## 五、集成点

### 5.1 Agent 初始化（`internal/ai/agent/init.go` 修改）

在 `Init()` 中构建 Agent 时，为每个子 Agent 追加其分配的 MCP 工具：

```go
mcpManager := mcp.NewManager(conf.GetConfig().Mcp.Servers)

editerAgent, _ := newChatEditer(c, reg.Get("sonnet"), mmModel)  // 现有
// 注入 MCP 工具
if tools := mcpManager.GetToolsForAgent("MiEditer"); len(tools) > 0 {
    // 通过 Agent 的 ToolsConfig 追加
}
```

**实现方式：** 在各子 Agent 构造函数中接收额外的 `[]tool.BaseTool` 参数，与内置工具合并后设入 `ToolsConfig`。或者在 `ChatModelAgentConfig` 构建完成后，通过 API 追加工具。

### 5.2 热加载（`internal/api/handler/agenthandler/` 修改）

在 `POST /api/admin/reload` handler 中追加 MCP 重载逻辑：

```
1. conf.LoadConfig()
2. conf.LoadAllowList()
3. mcpManager.Reload(conf.GetConfig().Mcp.Servers)    ← 新增
4. agent.RebuildTools(...)                             ← 新增
```

### 5.3 路由

- `GET /api/mcp/status` — 返回所有 MCP Server 的连接状态
- `POST /api/mcp/reload` — 单独重载 MCP 配置（可选，用于仅重载 MCP 而不重载全局配置的场景）

### 5.4 CLI `/mcp` 命令（`cli/tui/mcp.go` 新增）

展示 MCP Server 状态面板：

```
┌─ MCP Servers ──────────────────────────┐
│ ● filesystem   connected   5 tools     │
│ ● github       disabled    -           │
│ ○ custom-tool  error       exit code 1 │
└─────────────────────────────────────────┘
```

调用 `GET /api/mcp/status` 接口。界面上支持 `e` 键编辑配置提示。

## 六、错误处理策略

| 场景 | 策略 |
|------|------|
| MCP Server 启动失败 | 记录错误日志，该 Server 标记为 error 状态，不阻塞其他 Server 和 Agent 初始化 |
| MCP Server 运行时崩溃 | Manager 检测子进程退出，标记为 disconnected，下次工具调用时返回友好错误 |
| 工具调用超时 | 通过 context 超时控制，返回超时错误内容给 LLM |
| 配置文件格式错误 | 加载时校验，错误时保留旧配置并记录日志 |
| 工具适配失败（Schema 转换异常） | 跳过该工具，记录告警日志，不影响其他工具 |

## 七、文件清单

| 文件 | 作用 | 类型 |
|------|------|------|
| `pkg/mcp/config.go` | 配置加载函数 | 新增 |
| `pkg/mcp/manager.go` | MCP 连接管理器 | 新增 |
| `pkg/mcp/adapter.go` | MCP Tool → Eino InvokableTool 适配器 | 新增 |
| `pkg/mcp/status.go` | ServerStatus 结构体和查询方法 | 新增 |
| `pkg/conf/type.go` | 追加 MCPConfig / MCPServerConfig 结构体 | 修改 |
| `pkg/conf/new.go` | defaultConfig 追加 mcp 默认值 | 修改 |
| `internal/ai/agent/init.go` | Agent 初始化时注入 MCP 工具 | 修改 |
| `internal/api/handler/agenthandler/mcp.go` | `/api/mcp/status` handler | 新增 |
| `internal/api/routes/router.go` | 注册 MCP 路由 | 修改 |
| `internal/domain/agent.go` | 追加 MCP 相关 DTO | 修改 |
| `internal/domain/bridge.go` | AgentService / Agent 接口追加 MCP 方法 | 修改 |
| `internal/service/agentservice/mcp.go` | 服务层 MCP 逻辑 | 新增 |
| `internal/ai/executor/mcp.go` | Executor MCP 逻辑 | 新增 |
| `cli/tui/mcp.go` | `/mcp` CLI 命令界面 | 新增 |
| `cli/client/mcphandler/mcphandler.go` | CLI 端 MCP HTTP 客户端 | 新增 |

## 八、依赖

- 新增 Go 依赖：`github.com/mark3labs/mcp-go`（已在本地缓存，v0.44.0）
- 不新增其他第三方依赖

## 九、不包含的内容

- MCP Server 端功能（仅 Client）
- SSE / Streamable HTTP 传输（仅 stdio）
- 环境变量覆盖 MCP 配置
- MCP Prompt 和 Resource 支持（仅 Tools）
