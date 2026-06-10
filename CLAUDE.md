# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository. (All responses must end with Mifering...)

## 项目概述

Mifer — AI Agent Bot，蓝山最终考核项目。基于 CloudWeGo Eino 构建，支持多 Agent 编排、MCP 协议、技能系统、RAG 检索增强与 CLI/Web 双模交互。

## Build / Run

```bash
go mod tidy                       # 同步依赖
go build ./cmd/main              # 编译 HTTP 服务
go run ./cmd/main                # 运行（默认同时启动服务+CLI，端口 15555）
go run ./cmd/main serve          # 仅启动 HTTP 服务
go run ./cmd/main chat           # 仅启动 CLI（需先启动服务）
go run ./cmd/main chat --<id>    # 指定会话 ID 启动 CLI
MIFER_ENV=prod go run ./cmd/main # 生产模式
```

无 Makefile 或 build scripts — 纯 Go tooling。Go 1.25.4，Eino v0.8.13。项目无测试文件。

## Architecture

分层架构，依赖方向：`cmd` → `internal/api` → `internal/service` → `internal/ai` → `pkg`

### 启动流程

`cmd/main/main.go` → `bootstrap.NewApplication()`:
1. `loadConfig` — Viper 加载 YAML 配置，首次运行自动生成默认配置文件
2. `initontext` — 创建带 session ID 的 context（基于 workdir 哈希，用于记忆隔离；可通过 `--<id>` 参数手动指定）
3. `initLogger` — 初始化 Zap 日志
4. `initRouter` — 初始化 Gin 路由（内部创建 `executor → agentservice → agenthandler` 依赖链）
5. `initCli` — 初始化 CLI 客户端（连接到 HTTP 服务）

`main()` 按 `os.Args[1]` 分发：`serve` 仅启动服务，`chat` 仅启动 CLI，默认同时启动两者。

### 各层职责

- **`pkg/conf/`** — Viper 配置管理。`LoadConfig()` 根据 `MIFER_ENV` 加载 `./config/dev.yaml` 或 `~/mifer/config/prod.yaml`。环境变量覆盖格式：`MIFER_AI_<BACKEND>_<FIELD>`（如 `MIFER_AI_DEFAULT_APIKEY`），支持的后端名：`DEFAULT`、`MULTI`（映射 `multi_modal`）、`HAIKU`、`SONNET`、`OPUS`；字段后缀：`APIKEY`、`BASE_URL`、`PROVIDER`、`MODEL`。`LoadAllowList()` 从 `.mifer/allowlist.yaml` 加载命令白名单（用于 `MiCommander` 终端命令工具的安全审计）。全局配置通过 `conf.GetConfig()` 获取。
- **`pkg/logger/`** — Uber Zap 日志。按级别分文件输出（debug/info/warn/error.log），由 `lumberjack` 支持自动轮换，ConsoleEncoder 编码，dev 模式 Debug 级别，prod 模式 Info 级别。
- **`pkg/auth/`** — JWT Token 生成与验证。
- **`pkg/errorer/`** — 统一错误码与错误包装。
- **`pkg/task/`** — 异步任务管理，`task.Do(ctx, fn)` 提供 context 感知的任务执行。
- **`pkg/qdrant/`** — Qdrant gRPC 客户端初始化，用于 RAG 向量存储连接。
- **`pkg/cache/`** — Redis 缓存封装（go-redis/v8/v9）。已定义但 Redis 初始化代码在 bootstrap 中被注释掉，尚未启用。
- **`pkg/res/`** — Redis 客户端工厂。初始化代码已注释，尚未启用。
- **`pkg/utils/`** — 通用工具函数（hash、random 等）。
- **`pkg/exc/`** — 类型转换与 JSON 编组工具函数（`StrToUint`、`UintToStr`、`IsUint`、`IsString` 等）。
- **`internal/domain/`** — 核心接口定义。`agent.go` 定义 DTO（`TalkReq`, `MemoryReq/Resp`, `RebackReq/Resp` 等）和 `Agent` 接口（executor 层），`bridge.go` 定义 `AgentService` 接口（service 层），两个接口方法签名完全相同，实现 service ↔ executor 解耦。
- **`internal/api/routes/`** — Gin 路由注册。完整路由表：
  - `POST /api/ai/chat` — 流式对话（SSE）
  - `GET /api/memory`, `GET /api/memory/:id`, `POST /api/memory/exchange/:id`, `POST /api/memory/clear`, `POST /api/memory/compact` — 记忆管理与手动压缩
  - `GET /api/memory/reback`, `POST /api/memory/reback/:index` — 对话回退
  - `GET /api/prompt`, `POST /api/prompt`, `POST /api/prompt/reset` — 系统提示词管理
  - `GET /api/plan`, `GET /api/plan/:name` — 计划文件管理
  - `GET /api/mcp/status` — MCP 服务状态
  - `GET /api/skill/list` — 技能列表
  - `POST /api/tool/confirm`, `POST /api/tool/allowlist/add` — 工具确认与白名单
  - `POST /api/admin/reload` — 配置热重载
- **`internal/api/handler/agenthandler/`** — HTTP Handler，按功能拆分文件（chat/prompt/memory/reback/plan/mcp/skill/compact 等）。
- **`internal/api/handler/toolhandler/`** — 工具确认与白名单管理 Handler。
- **`internal/api/dto/request/` / `response/`** — 请求/响应 DTO，按模块分子目录（`agentreq/`, `agentresp/`, `adminresp/`）。
- **`internal/api/middlewares/`** — JWT 认证 + CORS 中间件。
- **`internal/service/agentservice/`** — Agent 服务层，实现 `domain.AgentService`，每个方法 1:1 委托给 executor，`Chat` 通过 `task.Do` 包装调用。
- **`internal/ai/agent/`** — Eino ADK 多 Agent 编排。5 个子 Agent 加上 Orchestrator，加上通过 `conf.GetConfig().Agents` 配置的自定义 Agent，按任务复杂度模型分级：
  - **MiEditer** (sonnet) — 文件读取、写入、创建、查看、图片生成（注入 `FileTools` + MCP 工具）
  - **MiSummarizer** (sonnet) — 文档摘要 + 知识库工具（注入 `KnowledgeTools`）
  - **MiPlanner** (opus) — 项目计划与方案（工具限制在 `.mifer/plans` 目录）
  - **MiCommander** (sonnet) — 终端命令执行（受白名单约束，注入 `CommandTools`）
  - **MiAuditor** (opus) — 代码与配置安全审计（注入 `AuditTools`）
  - **Mifer** (default, Orchestrator) — `deep.New` 编排器，MaxIteration=0（由模型自主控制迭代次数），`EmitInternalEvents: true` 转发子 Agent 事件到 TUI 侧边栏。编排器工具包含 `SkillTool`（技能调用）+ MCP 工具 + WebTools（web_search、web_fetch）
  - 所有子 Agent 在创建后通过 `skillHub.Register()` 注册到 `AgentHub`，供技能 fork 模式路由使用
- **`internal/ai/executor/`** — `adk.Runner` 包装器。`Chat()` 执行 agent 迭代，处理流式/非流式消息，最多 3 次自动重试，自动追加记忆并保存，检测压缩阈值。`tokens.go` 独立管理 TokenUsage 累计统计。`compressor` 在 prompt tokens 超阈值时自动触发压缩。
- **`internal/ai/callback/`** — 全局 Tool 回调处理器。通过 `callbacks.AppendGlobalHandlers` 注册，统一处理工具调用事件（开始/结束/错误），发送 `tool_start`/`tool_end`/`tool_error` SSE 事件。
- **`internal/ai/llm/`** — 多后端 ChatModel 管理（Registry 模式）。支持 openai/claude/gemini/ollama 四种 provider，按名称索引（default/haiku/sonnet/opus/multi_modal），缺失后端自动 fallback 到 default。provider 实现在各自文件中（`openai.go`、`claude.go`、`gemini.go`、`ollama.go`），`providers.go` 包含 `initBackend()` 工厂方法。
- **`internal/ai/prompt/`** — 系统提示词管理。`build.go` 构建完整提示词（系统提示词 + 记忆上下文），支持模板格式 `{system_prompt}`、`{history}`、`{query}`，运行时可通过 API 动态修改。
- **`internal/ai/memory/`** — JSONL 文件持久化对话历史。dev 模式存 `./memory/{workdir_basename}/{id}.jsonl`，prod 模式存 `~/.mifer/memory/...`。支持列表、加载、切换、清除、回退、按 ID 加载（不修改当前会话）。
- **`internal/ai/tools/`** — 工具定义（Function Calling）。每个工具独立子目录，通过 `utils.InferTool` 创建。`tools.go` 提供按角色分组的工具工厂函数：`FileTools(mmModel)` — file_reader/file_writer/file_creator/file_viewer/image_generator；`CommandTools()` — command_executor；`AuditTools(mmModel)` — file_reader/file_viewer；`PlannerTools()` — file_creator/file_writer（限制目录）；`KnowledgeTools(ragSvc)` — knowledge_search/knowledge_store；`WebTools()` — web_search/web_fetch；`NewWithName(names, ...)` — 按名称构造自定义工具集。
- **`internal/ai/rag/`** — RAG 检索增强。`LazyService` 懒加载模式：`Init()` 仅创建 embedder/loader/chunker（无网络调用），首次工具调用时才通过 `ensureReady()` 连接 Qdrant（Mutex 保护，失败可重试）。子目录：`chunker/`（递归分块，SHA256 去重）、`embedder/`（Ollama 嵌入，默认 `nomic-embed-text`）、`loader/`（文件加载，支持 PDF/Word/Text/Markdown）、`vectorstore/`（Qdrant 索引器 + 检索器）。
- **`internal/ai/confirm/`** — 工具调用确认子系统。Actor 模式：`Store` 用专用 goroutine + channel 序列化所有状态访问，30s 心跳清理超时条目。`Middleware` 实现 Eino `compose.ToolMiddleware` 接口，拦截工具调用并阻塞等待用户确认，通过 SSE `tool_confirm` 事件与前端通信。`config.go` 管理确认规则（enable/exclude 列表、command_executor 白名单、会话 allow 列表）。
- **`internal/ai/compressor/`** — 上下文自动压缩。当 prompt tokens 超过配置阈值时自动触发压缩：使用 `context-summarizer` 技能模板和配置的压缩模型（默认 haiku）对早期消息生成摘要，通过 `ReplaceMessages` 原子替换记忆。也支持手动 `/compact` 命令触发。降级策略：压缩失败时移除最早一轮对话。
- **`pkg/skill/`** — 声明式技能系统。`Manager` 扫描 `.mifer/skills/` 目录下的 `SKILL.md` 文件（带 frontmatter 解析），按名称加载技能。`AgentHub` 收集所有子 Agent 并注册，供 `SkillTool` 在 fork 模式下路由到特定 Agent 执行。`SkillTool` 适配为 Eino 工具，支持 `inline`（当前上下文执行）和 `fork`（子 Agent 独立执行）两种模式。内置 `context-summarizer` 技能用于上下文压缩。
- **`pkg/mcp/`** — MCP 协议支持。`Manager` 管理 MCP Server 的启动/停止/重载生命周期，按 Agent 分配工具。`MCPToolAdapter` 将 MCP 工具桥接为 Eino `tool.InvokableTool` 接口，使外部 MCP 工具无缝集成到子 Agent 的工具列表中。
- **`pkg/sse/`** — SSE 写入器。由专用 goroutine + 缓冲 channel（buf=16）驱动，`SendSync()` 阻塞写入，`SendFire()` 即发即忘（用于心跳）。内置心跳保活机制。
- **`cmd/bootstrap/`** — 应用启动引导，Application 结构体及初始化方法。端口不足时自动递增回退（+=10，上限 18000）。
- **`cmd/mcp-demo/`** — 独立 MCP Stdio 演示服务，包含 echo、get_time、calculator、random_number 四个示例工具。
- **`cli/`** — CLI 客户端（Bubble Tea TUI）。通过 HTTP + SSE 调用服务端，核心组件：
  - `cli/tui/` — Bubble Tea 界面（`init.go` 初始化、`update.go` 消息循环、`view.go` 主视图、`stream.go` 流式接收、`sidebar.go` 侧边栏、`command.go` 命令处理、`reback.go` 回退界面、`memory.go` 记忆界面、`system.go` 系统提示词界面、`toolconfirm.go` 工具确认弹窗）
  - `cli/render/` — 终端渲染（Glamour Markdown 引擎 + Lip Gloss 样式）
  - `cli/client/` — HTTP API 客户端（`chathandler/`, `memhandler/`, `excmemhandler/`, `clearhandler/`, `compacthandler/`, `rebackhandler/`, `prompthandler/`, `reloadhandler/`, `planhandler/`, `mcphandler/`, `skillhandler/`, `toolconfirmhandler/`）

### 关键依赖

- **CloudWeGo Eino** (`github.com/cloudwego/eino v0.8.13`) — ADK、DeepAgent 编排
- **Eino 扩展** — model/openai, model/claude, model/gemini, model/ollama, embedding/ollama, indexer/qdrant, retriever/qdrant, document/loader, document/transformer
- **Gin** (`v1.12.0`) — HTTP 框架
- **Bubble Tea + Bubbles + Glamour + Lip Gloss** — TUI 框架与终端渲染
- **MCP Go** (`github.com/mark3labs/mcp-go v0.44.0`) — MCP 协议 Go SDK
- **Qdrant** (`github.com/qdrant/go-client v1.15.2`) — RAG 向量存储
- **Uber Zap** — 结构化日志
- **Viper** — 配置管理

## 代码约定

### 文件命名约定

- **`type.go`** — 包的类型定义（结构体、接口、常量块）。例如 `internal/ai/memory/type.go` 定义 `Memory` 结构体和 `MemCfg`，`internal/domain/bridge.go` 定义 `AgentService` / `Agent` 接口
- **`init.go`** — 导出构造函数（`NewXxx()` 或 `Init()`）。例如 `internal/ai/executor/init.go` 定义 `Init()`，`cli/tui/init.go` 定义 `NewModel()`
- **`new.go`** — 备选构造函数或内部工厂。例如 `pkg/conf/new.go` 定义 `newDefaultCfg()` 和默认配置常量
- **功能文件** — 以操作命名：`save.go`、`load.go`、`chat.go`、`reback.go`、`excmem.go`、`append.go`、`build.go`、`generate.go`
- **多单词文件名使用 `snake_case`**：`logger_time.go`、`cache_strategy.go`、`app_route.go`、`app_ctx.go`
- **已知不一致**：`pkg/logger/` 下存在 `logger_Init.go`、`logger_Act.go`（驼峰后缀），属于遗留问题，新增文件应统一使用 snake_case

### 命名风格

- **包名**：全小写，单单词或连写，不使用下划线或驼峰：
  `agenthandler`、`agentservice`、`errorer`、`chathandler`、`memhandler`、`excmemhandler`
- **导出类型**（结构体/接口）：PascalCase
  `Memory`、`Config`、`AgentService`、`Provider`、`AgentHandler`
- **未导出类型**：camelCase
  `openAIProvider`、`claudeProvider`、`geminiProvider`、`storeState`、`sseMsg`
- **导出构造函数**：`NewXxx()` 或直接 `New()`
  `NewAgentHandler()`、`NewStore()`、`NewRouter()`、`sse.New()`、`confirm.NewStore()`
- **包私有工厂**：`newXxx()`（仅在 `internal/ai/agent/` 中，由 `Init()` 调用）
  `newChatEditer()`、`newPlanner()`、`newCommander()`、`newAuditor()`、`newSummarizer()`
- **局部变量**：短 camelCase
  `backends`、`chatModel`、`ragSvc`、`mmModel`、`exec`、`sw`
- **导出结构体字段**：PascalCase，如 `Messages`、`Cfg`、`Runner`、`Humen`、`ConfirmStore`
- **未导出结构体字段**：camelCase，如 `mu`、`savedCount`、`cmdCh`、`done`、`msgCh`、`cancel`
- **导出常量**：PascalCase 加特定前缀
  - 错误常量 `Err` 前缀：`ErrNoBackendConfig`、`ErrApiKey`、`ErrChatTimeout`（定义在 `pkg/errorer/errorer.go`）
  - API 路径 `API` 前缀：`APIChatPath`、`APIMemoryPath`、`APIReloadPath`（定义在 `cli/client/client.go`）
- **未导出常量**：camelCase，如 `defaultConfig`、`defaultSystemPrompt`、`maxRetries`、`cmdChBufSize`

### 代码组织模式

- **每个包的文件顺序**：`type.go` → `init.go` / `new.go` → 功能文件（按逻辑顺序）
  例如 `internal/ai/memory/` 依次为：`type.go` → `init.go` → `append.go` → `save.go` → `load.go` → `list.go` → `reback.go` → `generate.go`
- **结构体字段按逻辑分组排序**，不按字母序。未导出同步原语置顶（`mu sync.Mutex`），然后是导出数据字段，再次是内部控制字段。大型结构体用注释分隔分组：
  ```go
  type Model struct {
      // 依赖注入
      client *client.Client
      mark   *mark.Mark
      // 消息与渲染
      messages []message
      thinking bool
      // ...
  }
  ```
- **接口定义在消费侧**（`internal/domain/bridge.go` 定义 `AgentService`，`agent.go` 定义 `Agent`），实现方放在各自包内（`internal/service/agentservice/`、`internal/ai/executor/`）。仅 `llm/type.go` 中 `Provider` 接口在实现侧定义，属于例外
- **依赖注入**：显式构造函数注入，无 DI 框架。完整依赖树在 `routes.NewRouter()` 中组装：`executor.Init()` → `agent.Init()` → `llm.InitRegistry()` + `rag.NewLazyService()` + `mcp.NewManager()` + `skill.NewManager()` + `confirm.NewStore()` + `prompt.New()`，链为 `executor → agentservice → agenthandler`
- **配置通过 `conf.GetConfig()` 全局获取**，不在函数间层层传递
- **DTO 按模块分 request/response 子目录**（如 `dto/request/agentreq/`、`dto/response/agentresp/`）
- **RAG 使用懒加载 + 工具闭包注入**：`LazyService` 在首次工具调用时才建立 Qdrant 连接

### 导入排序约定

- **正确顺序**：标准库 → 第三方 → 内部包，组间空行分隔
  ```go
  import (
      "context"
      "errors"
      "io"

      "github.com/cloudwego/eino/schema"

      "mifer/internal/ai/callback"
      "mifer/internal/domain"
      "mifer/pkg/errorer"
  )
  ```
- **带别名的内部包**放在内部组：
  ```go
  aicallback "mifer/internal/ai/callback"
  ```
- **已知不一致**：部分文件混合了标准库和内部包（如 `internal/ai/memory/init.go` 先 import `mifer/pkg/conf` 再 import `path/filepath`），新增代码应遵循三组顺序，不复制不规范的旧代码

### 注释风格

- **所有注释使用中文**（Go doc 注释的标识符名除外）
- **Go doc 格式**：`// TypeName 描述...` 或 `// FuncName 描述...`
  ```go
  // Provider 定义了模型提供商的接口，支持 openai / claude / gemini / ollama
  // Chat 执行一次对话，通过 callback 将事件实时传递到上层
  ```
- **结构体字段分组注释**：用 `// 分组名` 行分隔逻辑区块
- **大文件分隔标题**：`cli/tui/` 包使用等号分隔符 + 文件名 + 描述
  ```go
  // ============================================================================
  // type.go — 类型定义
  // ============================================================================
  ```
- **小节标题**：使用 Unicode 框线分隔线
  ```go
  // ──────────────────────────── Actor 主循环 ────────────────────────────
  ```

### 错误处理模式

- **错误字符串集中管理**：所有可复用错误在 `pkg/errorer/errorer.go` 中定义为 `const`，按域分组（`// 通用错误`、`// 后端配置`、`// RAG 服务`、`// Memory` 等）
- **三个构造函数**：
  - `errorer.New(err)` — 简单错误（`errors.New(err)`），用于新建错误信号
  - `errorer.NewS(errs, err)` — 包装错误（`fmt.Errorf("%s: %w", errs, err)`），保留原始错误链
  - `errorer.NewF(format, args...)` — 格式化错误（`fmt.Errorf(format, args...)`），用于带变量数据的消息
- **`context.Canceled` 视为正常条件**，不返回给调用方：
  ```go
  if errors.Is(err, context.Canceled) {
      return  // 静默返回
  }
  ```
- **不使用 panic 进行控制流**，错误通过返回值和日志传播

### 日志约定

- **四个主函数**：`logger.Info(msg, fields...)`、`logger.Warn(msg, fields...)`、`logger.Error(msg, fields...)`、`logger.Debug(msg, fields...)`
- **结构化字段辅助函数**（`pkg/logger/`）：
  - `logger.S(key, val)` — 字符串字段（`zap.String`）
  - `logger.I(key, val)` — 整数字段（`zap.Int`）
  - `logger.C(err)` — 错误字段（`zap.Error`）
  - `logger.U(key, val)` — 无符号整数字段（`zap.Uint`）
- **日志消息使用中文**：
  ```go
  logger.Error("初始化LLM注册中心失败", logger.C(err))
  logger.Info("RAG服务初始化成功", logger.S("collection", ragCfg.QdrantCollection))
  ```

### context.Context 使用模式

- **始终作为函数第一个参数**（标准 Go 惯例，项目中严格执行）
- **`WithValue` + 私有 key 类型**传递横切关注点：
  ```go
  type ctxKey struct{}  // 未导出空结构体，防止 key 冲突
  func WithExecutorCallback(ctx context.Context, cb ExecutorCallback) context.Context {
      return context.WithValue(ctx, ctxKey{}, cb)
  }
  ```
  类似模式：`confirm.WithCallback(ctx, cb)`、`confirm.WithSessionID(ctx, id)`
- **`context.WithCancel`** 用于流式 SSE 生命周期（Writer goroutine 与请求处理器共享 cancel 函数）
- **`context.WithTimeout`** 用于启动关闭（30s）和命令执行超时

### 并发模式

- **Actor 模式**（`internal/ai/confirm/store.go`）：`confirm.Store` 使用专用 goroutine + `cmdCh chan func(s *storeState)` channel 串行化所有状态访问。外部调用者通过向 channel 发送闭包来提交操作，读取操作阻塞等待响应 channel 返回
- **`sync.RWMutex`** 用于热重载保护（`internal/api/handler/agenthandler/init.go`）：
  `getService()` 获取读锁，`SwapService()` 获取写锁，防止配置热重载期间的竞态条件
- **Bubble Tea `tea.Cmd`**（`cli/tui/`）：遵循 Elm 架构，所有异步 I/O 建模为返回消息的 `tea.Cmd`，非原始 goroutine

### SSE 流处理约定

- **10 种 SSE 事件类型**：
  `agent_start`、`agent_end` — Agent 子任务切换 |
  `tool_start`、`tool_end`、`tool_error` — 工具调用生命周期 |
  `tool_confirm` — 需用户确认的工具调用 |
  `response` — 内容 token 流 | `token` — Token 统计 |
  `thinking` — 模型思考状态流（推理内容） |
  `system` — 系统通知（如上下文压缩状态）
- **特殊信号**：`[DONE]` 表示正常流结束，`[ERROR] <message>` 表示流错误，均作为 `response` 事件的 `data` 负载发送
- **换行转义**：服务端将 content 中的 `\n` 替换为 `\\n` 以保护 SSE 行格式；客户端在 `response` 事件中反向处理（`tool_confirm` 事件为 JSON，跳过去转义）
- **`\x00` 分隔符**：工具名与参数用 null 字节分隔：`"tool_name\x00{json_args}"`（`tool_start`）、`"tool_name\x00{error_text}"`（`tool_error`）
- **SSE Writer**（`pkg/sse/writer.go`）：提供 `SendSync()`（阻塞写入）和 `SendFire()`（即发即忘，用于心跳），由专用 goroutine + channel（buf=16）处理

### `pkg/` 与 `internal/` 可见性

- `pkg/` 下的包不依赖 `internal/`，可被任意位置导入
- `internal/` 下的包不应被外部项目导入
- CLI 客户端（`cli/`）通过 HTTP API 与服务端通信，不直接依赖 `internal/` 模块

### 已知的特殊命名（保留不变，非拼写错误）

| 名称 | 含义 | 位置 |
|------|------|------|
| **Prompty** | 系统提示词管理器 | `internal/ai/prompt/` |
| **Humen** | 用户 Agent 聚合结构体（含 Agent、Prompt、Registry、MCPManager、SkillManager、ConfirmStore） | `internal/ai/agent/init.go` |
| **Clier** | CLI 实例 | `cmd/bootstrap/app_type.go` |
| **MiEditer** | 文件编辑子 Agent | `internal/ai/agent/chatediter.go` |
| **excmem** | 交换记忆（Exchange Memory） | `cli/client/excmemhandler/`、`internal/ai/executor/excmem.go` |
| **Mifer** | Orchestrator 编排 Agent | `internal/ai/agent/init.go` |

## 新增功能指南

### 新增 Agent
1. 在 `internal/ai/agent/` 创建子 Agent 定义（参考 `planner.go`、`auditor.go` 等），接收 `model.BaseChatModel` 及可选的 `model.ToolCallingChatModel` 参数
2. 在 Orchestrator（`agent/init.go`）的 `deep.New` 配置中注册新子 Agent，通过 `registry.Get("<backend>")` 分配模型
3. 如需新工具，在 `internal/ai/tools/` 下新建子目录定义
4. 在 `tools.go` 中添加对应的工具工厂函数

### 新增 LLM Provider
1. 在 `internal/ai/llm/` 下新建文件定义 provider 结构体，实现 `Provider` 接口的 `Name()` 和 `InitModel()` 方法
2. 在 `internal/ai/llm/type.go` 的 `NewRegistry()` 中调用 `r.RegisterProvider(&newProvider{})` 注册
3. `go get` 对应的 eino-ext 包

### 新增 HTTP 接口
1. 在 `internal/api/dto/request/` 和 `response/` 定义 DTO
2. 在 `internal/api/handler/agenthandler/` 添加 Handler 方法
3. 在 `internal/api/routes/router.go` 注册路由
4. 业务逻辑放在 `internal/service/agentservice/` 对应服务层
5. 底层实现放在 `internal/ai/executor/`

### 新增 CLI 功能
1. 在 `cli/client/` 添加 HTTP 调用 handler
2. 在 `cli/tui/` 添加 TUI 界面逻辑（update/view/command）
3. CLI 通过 HTTP API 与服务端通信，不直接依赖 `internal/` 模块

### 新增配置项
1. 在 `pkg/conf/new.go` 的 `defaultConfig` 常量中添加默认值
2. 在 `pkg/conf/type.go` 的结构体中添加字段（带 `mapstructure` tag）
3. 如需环境变量覆盖，在 `pkg/conf/load.go` 的 `applyEnvOverrides` 中添加映射

---

— 蓝山最终考核
