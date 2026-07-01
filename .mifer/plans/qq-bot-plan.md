# Mifer QQ Bot 实现计划

## 一、架构原理

### 1.1 核心原则：QQ adapter 是 HTTP 客户端，不是服务端组件

CLI 和 Web 是 Mifer HTTP API 的客户端，QQ adapter 同理。**QQ 不 import 任何 `internal/` 下的代码**，仅通过 HTTP + SSE 与服务端通信。

```
                                Mifer 进程
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                 HTTP API Server                       │   │
│  │  POST /api/ai/chat           ← SSE 流式对话           │   │
│  │  POST /api/memory/exchange/:id ← 切换记忆会话         │   │
│  │  POST /api/tool/confirm      ← 确认/拒绝工具调用      │   │
│  │  （现有 API，一个不改）                                │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │ HTTP + SSE                         │
│         ┌───────────────┼───────────────┐                    │
│         ▼               ▼               ▼                    │
│  ┌────────────┐ ┌────────────┐ ┌───────────────┐            │
│  │  CLI (现有) │ │  Web (现有) │ │ QQAdapter(新) │            │
│  │  终端 TUI   │ │  浏览器前端 │ │  qq/          │            │
│  │  cli/       │ │             │ │  HTTP 客户端   │            │
│  └────────────┘ └────────────┘ └───────┬───────┘            │
│                                        │ WebSocket           │
└────────────────────────────────────────┼─────────────────────┘
                                         ▼
                                  ┌────────────┐
                                  │  SnowLuma  │
                                  │ (OneBot v11)│
                                  └─────┬──────┘
                                        │ DLL 注入
                                        ▼
                                  ┌────────────┐
                                  │  QQ 客户端  │
                                  └────────────┘
```

### 1.2 QQ 消息处理全链路

```
QQ 消息到达
    │
    ▼
SnowLuma WS → QQAdapter 收到 OneBot 事件
    │
    ├── 1. CQ 码清洗 + @ 检测
    │
    ├── 2. POST /api/memory/exchange/qq_private_12345   ← 切换记忆会话
    │
    ├── 3. POST /api/ai/chat { "content": "..." }       ← SSE 流式对话
    │       │
    │       ├── SSE "response"     → 收集回复文本
    │       ├── SSE "tool_confirm" → 解析 → POST /api/tool/confirm 自动响应
    │       └── SSE "[DONE]"       → 对话结束
    │
    └── 4. 回复文本 → HTTP POST OneBot API send_msg → QQ
```

### 1.3 三层职责划分

```
┌─────────────────────────────────────────────────────────┐
│ 层                  职责                  import 范围    │
├─────────────────────────────────────────────────────────┤
│ qq/              QQ 消息通道客户端       pkg/conf       │
│                  WS↔HTTP 桥接           pkg/logger     │
│                  无 AI 逻辑             gorilla/ws     │
│                                        net/http       │
├─────────────────────────────────────────────────────────┤
│ internal/ai/     Agent 工具（供 LLM     pkg/conf       │
│ tools/qq/         Function Calling）    pkg/logger     │
│                                        eino/tool      │
│                                        net/http       │
├─────────────────────────────────────────────────────────┤
│ cmd/bootstrap/   组装层：创建 Sender、   pkg/*          │
│ app_qq.go        启动 QQAdapter         internal/*     │
│                                         qq             │
└─────────────────────────────────────────────────────────┘
```

**关键**：`qq/` 和 `internal/ai/tools/qq/` 互不知晓对方存在，通过 bootstrap 组装。

---

## 二、文件夹层次

```
mifer/
├── cli/                                       ← （现有，不改）
│   ├── tui/
│   ├── client/
│   └── render/
│
├── qq/                                        ← 新：QQ 消息通道客户端（HTTP 消费者）
│   ├── type.go                                ←   QQAdapter 结构体、配置、事件类型
│   ├── adapter.go                             ←   核心：事件循环 → HTTP API → 回复
│   ├── ws_client.go                           ←   SnowLuma WebSocket 连接
│   ├── mifer_client.go                        ←   Mifer HTTP API 客户端（chat/memory/confirm）
│   ├── onebot_client.go                       ←   OneBot HTTP API 客户端（send_msg）
│   └── parser.go                              ←   CQ 码清洗 + @ 检测
│
├── internal/
│   ├── ai/
│   │   ├── memory/
│   │   │   ├── validate.go                    ← 新：validateID + buildFilePath
│   │   │   ├── load.go                        ← 改：使用 validateID + buildFilePath
│   │   │   ├── save.go                        ← 改：同上
│   │   │   ├── type.go                        ← 改：ReplaceMessages 使用 buildFilePath
│   │   │   ├── reback.go                      ← 改：Reback 使用 buildFilePath
│   │   │   └── list.go                        ← 改：递归扫描
│   │   ├── tools/
│   │   │   ├── tools.go                       ← 改：新增 QQTools() 工厂
│   │   │   └── qq/                            ← 新：QQ 工具（Agent Function Calling）
│   │   │       └── send_message.go            ←   qq_send_message + Sender 接口
│   │   └── agent/
│   │       └── init.go                        ← 改：Mifer 编排器工具列表追加 QQTools()
│   ├── api/                                   ← （全部不改）
│   ├── service/                               ← （全部不改）
│   └── domain/                                ← （全部不改）
│
├── pkg/
│   └── conf/
│       ├── type.go                            ← 改：新增 QQConfig / QQOnebotConfig / QQBotConfig
│       ├── new.go                             ← 改：默认配置追加 qq 段
│       └── load.go                            ← 不改（环境变量覆盖可选）
│
├── cmd/
│   └── bootstrap/
│       ├── app_qq.go                          ← 新：组装 QQ adapter + 注入 Sender
│       ├── app_type.go                        ← 改：Application 新增 qqAdapter 字段
│       ├── app_run.go                         ← 改：启动流程中调用 initQQ()
│       └── app_down.go                        ← 改：Shutdown 中停止 QQ adapter
│
└── go.mod                                     ← 改：新增依赖 gorilla/websocket
```

---

## 三、各模块详细设计

### 3.1 `internal/ai/memory/validate.go` — 记忆系统层级路径

当前问题：4 个文件中的 `strings.Contains(id, "/")` 拒绝层级路径。

```go
package memory

// validateID 验证会话 ID。允许 "12345" 和 "qq_private/12345"，拒绝 ".."、绝对路径、空值。
func validateID(id string) error

// buildFilePath 构建 JSONL 完整路径，自动创建嵌套子目录。
func buildFilePath(memPath, id string) (string, error)
```

| 文件 | 改动 |
|------|------|
| `load.go:19` | `strings.Contains` → `validateID(cfg.Id)` |
| `load.go:28` | `filepath.Join(MemPath, id+".jsonl")` → `buildFilePath(MemPath, id)` |
| `save.go:18` | 同上 |
| `save.go:31` | 同上 |
| `type.go:50` | `ReplaceMessages` 中使用 `buildFilePath` |
| `reback.go:74` | `Reback` 中使用 `buildFilePath` |
| `list.go` | `os.ReadDir` → `filepath.WalkDir`，返回相对路径作为 ID |

生成的文件结构：

```
memory/A/
├── 8374629150.jsonl                  ← 现有平坦 ID
├── qq_private/
│   ├── 12345.jsonl                   ← 用户 12345 的私聊记忆
│   └── 67890.jsonl                   ← 用户 67890 的私聊记忆
└── qq_group/
    └── 789/
        ├── 12345.jsonl               ← 群 789 中用户 12345 的记忆
        └── 67890.jsonl               ← 群 789 中用户 67890 的记忆
```

---

### 3.2 `qq/` — QQ 消息通道客户端（HTTP 消费者）

#### `type.go`

```go
package qq

// Config QQ adapter 配置（从 conf.QQConfig 映射而来，不 import internal）
type Config struct {
    WsURL          string
    MiferURL       string   // Mifer HTTP 服务地址，如 "http://127.0.0.1:15555"
    OnebotHttpURL  string
    OnebotToken    string
    BotQQ          int64
    GroupReplyMode string   // "mention_only" | "always"
    PrivateEnabled bool
}

// QQAdapter 持有各子组件和生命周期 context。
type QQAdapter struct {
    cfg    Config
    ws     *wsClient
    mifer  *miferClient
    onebot *onebotClient
    ctx    context.Context
    cancel context.CancelFunc
}

// oneBotEvent 仅包含实际使用的 JSON 字段（包私有）。
type oneBotEvent struct {
    PostType    string      `json:"post_type"`
    MessageType string      `json:"message_type"`
    UserID      int64       `json:"user_id"`
    GroupID     int64       `json:"group_id"`
    Message     string      `json:"message"`
    RawMessage  string      `json:"raw_message"`
    Sender      eventSender `json:"sender"`
}

type eventSender struct {
    Nickname string `json:"nickname"`
}
```

#### `adapter.go`

```go
package qq

// NewAdapter 创建 QQ adapter。所有外部依赖通过 Config 注入。
func NewAdapter(cfg Config) *QQAdapter

// Start 启动：连接 SnowLuma WS → 进入事件循环。应在 goroutine 中调用。
func (a *QQAdapter) Start() error

// Stop 停止：关闭 WS 连接 + 取消 context。
func (a *QQAdapter) Stop()

// ── 内部流程 ──

// eventLoop()              从 ws.events() 取事件，调用 handleMessage()
// handleMessage(event)     按 MessageType 分发 → handlePrivate / handleGroup
// handlePrivate(event)     buildSessionID → cleanCQ → processAndReply
// handleGroup(event)       isAtBot 检查 → buildSessionID → cleanCQ → processAndReply
// processAndReply(sid, event, query)
//   1. mifer.ExchangeMemory(sid)
//   2. mifer.Chat(query, callback)     ← SSE 流
//        callback 处理:
//          "response"     → replyBuf.WriteString
//          "tool_confirm" → mifer.ConfirmTool(data)   ← 自动响应
//   3. onebot.SendReply(event, replyBuf.String())
```

调用链（每个函数都有明确调用者，无死代码）：

```
NewAdapter → Start → eventLoop → handleMessage
                                   ├── handlePrivate → processAndReply
                                   └── handleGroup   → processAndReply
                                                          ├── mifer.ExchangeMemory
                                                          ├── mifer.Chat
                                                          │     └── callback → mifer.ConfirmTool
                                                          └── onebot.SendReply
Stop → ws.stop
```

#### `ws_client.go`

```go
package qq

// wsClient SnowLuma WebSocket 事件读取器。
type wsClient struct {
    url     string
    token   string
    eventCh chan *oneBotEvent   // buf=64
    conn    *websocket.Conn
    ctx     context.Context
    cancel  context.CancelFunc
}

// 方法：connect() error, stop(), events() <-chan *oneBotEvent
// 重连：指数退避 1s → 2s → 4s → ... → 60s（上限），连接成功重置

// 依赖：github.com/gorilla/websocket
```

#### `mifer_client.go`

```go
package qq

// miferClient Mifer HTTP API 客户端。
type miferClient struct {
    baseURL string
    client  *http.Client
}

// ExchangeMemory(sessionID string) error
//   POST {baseURL}/api/memory/exchange/{sessionID}

// Chat(query string, callback func(event, data string) error) error
//   POST {baseURL}/api/ai/chat  body: {"content": query}
//   读取 SSE 流，逐事件调用 callback
//   支持事件: response, tool_confirm, agent_start, agent_end, token, thinking, system

// ConfirmTool(eventData string) error
//   解析 tool_confirm 事件中的 confirm_id 和 tool_name
//   若 tool_name 在 allowedTools 中 → POST {baseURL}/api/tool/confirm {"id":..., "action":"confirm"}
//   否则 → POST {baseURL}/api/tool/confirm {"id":..., "action":"deny"}
```

> **注**：`allowedTools` 在 `adapter.go` 中硬编码为 `{"qq_send_message": true}`，只允许 QQ 消息发送工具自动通过，其余一律拒绝。后续如需开放更多工具，在此 map 追加。

#### `onebot_client.go`

```go
package qq

// onebotClient OneBot HTTP API 客户端。
type onebotClient struct {
    httpURL string
    token   string
    client  *http.Client
}

// SendPrivateMsg(userID int64, message string) error
// SendGroupMsg(groupID int64, message string) error
// SendReply(event *oneBotEvent, content string)  ← 根据 MessageType 分发

// callAPI(action string, params map[string]interface{}) error
//   POST {httpURL}/{action}
//   Authorization: Bearer {token}
//   Body: {"action":..., "params":...}
```

#### `parser.go`

```go
package qq

// cleanCQ(raw string) string
//   [CQ:at,qq=xxx] → 移除（isAtBot 独立处理）
//   [CQ:image,...]  → "[图片]"
//   [CQ:face,...]   → "[表情]"
//   其他 CQ 码      → 移除

// isAtBot(message string, botQQ int64) bool
//   检查 message 中是否包含 [CQ:at,qq={botQQ}]
```

---

### 3.3 `internal/ai/tools/qq/send_message.go` — Agent 工具

```go
package qq

// Sender QQ 消息发送器接口。QQ adapter 初始化时由 bootstrap 注入实现。
// 未注入时工具返回友好错误，Agent 自然理解 QQ 不可用。
var Sender interface {
    SendPrivateMsg(userID int64, message string) error
    SendGroupMsg(groupID int64, message string) error
}

// New 创建 qq_send_message 工具。
func New() (tool.InvokableTool, error) {
    return utils.InferTool("qq_send_message",
        "发送 QQ 消息。target_type: private(私聊) / group(群聊)",
        func(ctx context.Context, input struct {
            TargetType string `json:"target_type" jsonschema:"required,description=private 或 group"`
            TargetID   int64  `json:"target_id"   jsonschema:"required,description=QQ号或群号"`
            Content    string `json:"content"     jsonschema:"required,description=消息内容"`
        }) (string, error) {
            if Sender == nil {
                return "", fmt.Errorf("QQ 消息服务未启用")
            }
            // ...
        },
    )
}
```

---

### 3.4 `cmd/bootstrap/app_qq.go` — 组装层

```go
package bootstrap

func (app *Application) initQQ() error {
    cfg := conf.GetConfig().QQ
    if !cfg.Enabled || cfg.Bot.QQ == 0 {
        return nil
    }

    // 1. 创建 OneBot HTTP 消息发送器，注入到工具包
    qqtools.Sender = &onebotSender{
        httpURL: cfg.Onebot.HttpURL,
        token:   cfg.Onebot.AccessToken,
    }

    // 2. 创建 QQ adapter（HTTP 客户端，不依赖 internal）
    adapter := qq.NewAdapter(qq.Config{
        WsURL:          cfg.Onebot.WsURL,
        MiferURL:       fmt.Sprintf("http://127.0.0.1:%d", conf.GetConfig().Gin.Port),
        OnebotHttpURL:  cfg.Onebot.HttpURL,
        OnebotToken:    cfg.Onebot.AccessToken,
        BotQQ:          cfg.Bot.QQ,
        GroupReplyMode: cfg.Bot.GroupReplyMode,
        PrivateEnabled: cfg.Bot.PrivateEnabled,
    })
    app.qqAdapter = adapter
    go func() { _ = adapter.Start() }()

    logger.Info("QQ Bot 已启动", logger.I("qq", int(cfg.Bot.QQ)))
    return nil
}

// onebotSender 实现 qqtools.Sender 接口，直接 HTTP 调用 OneBot API。
type onebotSender struct {
    httpURL string
    token   string
}
func (s *onebotSender) SendPrivateMsg(userID int64, message string) error { ... }
func (s *onebotSender) SendGroupMsg(groupID int64, message string) error { ... }
```

---

### 3.5 工具注入

```go
// internal/ai/tools/tools.go
func QQTools() []tool.BaseTool { ... }

// internal/ai/agent/init.go — Mifer 编排器 Tools 列表追加：
tools = append(tools, tools.QQTools()...)
```

---

### 3.6 配置

**`pkg/conf/type.go`** — 新增：

```go
type Config struct {
    // ... 现有字段 ...
    QQ QQConfig `mapstructure:"qq"`
}

type QQConfig struct {
    Enabled bool           `mapstructure:"enabled"`
    Onebot  QQOnebotConfig `mapstructure:"onebot"`
    Bot     QQBotConfig    `mapstructure:"bot"`
}

type QQOnebotConfig struct {
    WsURL       string `mapstructure:"ws_url"`
    HttpURL     string `mapstructure:"http_url"`
    AccessToken string `mapstructure:"access_token"`
}

type QQBotConfig struct {
    QQ             int64  `mapstructure:"qq"`
    GroupReplyMode string `mapstructure:"group_reply_mode"` // "mention_only" | "always"
    PrivateEnabled bool   `mapstructure:"private_enabled"`
}
```

**`pkg/conf/new.go`** — 默认配置：

```yaml
qq:
  enabled: false
  onebot:
    ws_url: "ws://127.0.0.1:3001"
    http_url: "http://127.0.0.1:3001"
    access_token: ""
  bot:
    qq: 0
    group_reply_mode: "mention_only"
    private_enabled: true
```

---

## 四、改动文件总览

| 层 | 文件 | 操作 | 改动量 |
|----|------|------|--------|
| **qq/** | type.go | 新 | ~50 行 |
| | adapter.go | 新 | ~120 行 |
| | ws_client.go | 新 | ~100 行 |
| | mifer_client.go | 新 | ~100 行 |
| | onebot_client.go | 新 | ~60 行 |
| | parser.go | 新 | ~30 行 |
| **internal/ai/memory/** | validate.go | 新 | ~30 行 |
| | load.go | 改 | 4 行 |
| | save.go | 改 | 4 行 |
| | type.go | 改 | 3 行 |
| | reback.go | 改 | 3 行 |
| | list.go | 改 | ~20 行 |
| **internal/ai/tools/qq/** | send_message.go | 新 | ~40 行 |
| **internal/ai/tools/** | tools.go | 改 | +10 行 |
| **internal/ai/agent/** | init.go | 改 | +1 行 |
| **pkg/conf/** | type.go | 改 | +20 行 |
| | new.go | 改 | +10 行 |
| **cmd/bootstrap/** | app_qq.go | 新 | ~60 行 |
| | app_type.go | 改 | +1 行 |
| | app_run.go | 改 | +3 行 |
| | app_down.go | 改 | +3 行 |
| **go.mod** | | 改 | +1 依赖 |

**统计**：8 个新文件 + 12 个改动文件，新增约 660 行，改动约 60 行。

### 不改的部分（明确边界）

| 模块 | 原因 |
|------|------|
| HTTP API（routes / handler） | 现有 API 已满足需求，不加新端点 |
| Router | 不暴露内部服务，QQ adapter 不经过它 |
| Executor / Agent / Service | QQ adapter 通过 HTTP 调用，不 import |
| Confirm 中间件 | QQ adapter 通过 `/api/tool/confirm` 接口自动响应 |
| conf/load.go | 环境变量覆盖可选，MVP 不实现 |

---

## 五、工具确认流程

```
Agent 调用 qq_send_message 工具
    │
    ▼
Confirm 中间件拦截 → 创建确认项 → SSE "tool_confirm" 事件
    │
    ▼
QQAdapter.miferClient 的 SSE callback 收到 "tool_confirm"
    │
    ├── 解析 eventData → { confirm_id, tool_name }
    ├── tool_name == "qq_send_message" → POST /api/tool/confirm { "id": confirm_id, "action": "confirm" }
    └── 其他 tool_name               → POST /api/tool/confirm { "id": confirm_id, "action": "deny" }
```

**三个通道共用同一套 confirm 系统**：

| 通道 | 确认方式 |
|------|---------|
| CLI | SSE `tool_confirm` → TUI 弹窗 → 用户按键 → POST `/api/tool/confirm` |
| Web | SSE `tool_confirm` → 前端弹窗 → 用户点击 → POST `/api/tool/confirm` |
| QQ | SSE `tool_confirm` → QQ adapter 回调 → 代码判断 → POST `/api/tool/confirm` |

区别仅在**谁调用** `/api/tool/confirm`——人工还是代码。中间件和 Store 不做任何修改。

---

## 六、热重载兼容性

```
POST /api/admin/reload
    │
    ▼
Router.Reload() → executor.Init() → 新 AgentService → agentHandler.SwapService()
    │
    ▼
QQ adapter 不受影响 —— 它通过 HTTP 调用 /api/ai/chat，下次请求自然使用新 service
```

**无需任何特殊处理**。QQ adapter 作为 HTTP 客户端，与服务端的热重载完全解耦。

---

## 七、实现顺序

```
Phase 1  ═══ 记忆系统层级路径（0.5 天） ═══
         □ 新建 memory/validate.go
         □ 改造 load.go / save.go / type.go / reback.go（各 2-4 行）
         □ 改造 list.go（WalkDir）
         □ 验证：创建层级 ID 会话并读写

Phase 2  ═══ QQ adapter 客户端（2 天） ═══
         □ qq/type.go
         □ qq/ws_client.go（WebSocket + 重连）
         □ qq/mifer_client.go（HTTP + SSE 解析 + confirm 自动响应）
         □ qq/onebot_client.go（OneBot HTTP API）
         □ qq/parser.go（CQ 码清洗 + @ 检测）
         □ qq/adapter.go（事件循环 + 消息路由）
         □ 验证：ws_client 连接 SnowLuma 收到事件

Phase 3  ═══ 工具 + 配置 + 组装（1 天） ═══
         □ internal/ai/tools/qq/send_message.go（工具 + Sender 接口）
         □ internal/ai/tools/tools.go（QQTools 工厂）
         □ internal/ai/agent/init.go（+1 行）
         □ pkg/conf/type.go + new.go
         □ cmd/bootstrap/app_qq.go（Sender 注入 + adapter 创建）
         □ cmd/bootstrap/app_type.go / app_run.go / app_down.go
         □ 验证：全链路收发消息

Phase 4  ═══ 联调（1.5 天） ═══
         □ 启动 SnowLuma + QQ
         □ 私聊对话 + 记忆隔离验证
         □ 群聊 @ 对话 + 记忆隔离验证
         □ Agent 主动调用 qq_send_message 工具
         □ 断线重连
         □ 热重载（reload 后 QQ 通道正常）

─────────────────────────────────────────
总计：5 个工作日，8 个新文件 + 12 个改动文件
```

---

## 八、明确不做

| 项目 | 原因 |
|------|------|
| notice / request 事件处理 | post_type=message 之外的静默消费 |
| heartbeat | ws_client 收到后 ack，不上抛 |
| 多 Bot | 单 Bot 足够 |
| 速率限制 | MVP 不需要 |
| 主动推送 | 仅被动回复 |
| 富文本回复 | 纯文本发送，不构造 CQ 码 |
| QQ HTTP API 新增 | 复用现有 `/api/memory` `/api/ai/chat` `/api/tool/confirm` |
| internal/ 暴露 service | QQ adapter 不 import internal |
| 新增 confirm 机制 | 通过 HTTP `/api/tool/confirm` 复用现有系统 |

---

## 九、新增依赖

```
github.com/gorilla/websocket    ← WebSocket 客户端（qq/ 使用）
```

无其他新依赖。`qq/` 的 HTTP 调用使用标准库 `net/http`。
