# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Mifer — AI Agent Bot，蓝山最终考核项目。基于 CloudWeGo Eino 构建，支持多 Agent 编排、MCP 协议、技能系统、RAG 检索增强与 CLI/Web 双模交互。

## Build / Run

```bash
go mod tidy                       # 同步依赖
go build ./cmd/main              # 编译 HTTP 服务
go run ./cmd/main                # 运行（默认同时启动服务+CLI，端口 8080）
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
2. `initContext` — 创建带 session ID 的 context（基于 workdir 哈希，用于记忆隔离；可通过 `--<id>` 参数手动指定）
3. `initLogger` — 初始化 Zap 日志
4. `initRouter` — 初始化 Gin 路由（内部创建 `executor → agentservice → agenthandler` 依赖链）
5. `initCli` — 初始化 CLI 客户端（连接到 HTTP 服务）

`main()` 按 `os.Args[1]` 分发：`serve` 仅启动服务，`chat` 仅启动 CLI，默认同时启动两者。

### 各层职责

- **`pkg/conf/`** — Viper 配置管理。`LoadConfig()` 根据 `MIFER_ENV` 加载 `./config/dev.yaml` 或 `~/.mifer/config/prod.yaml`。环境变量覆盖格式：`MIFER_AI_<BACKEND>_<FIELD>`（如 `MIFER_AI_DEFAULT_APIKEY`）。`LoadAllowList()` 从 `.mifer/allowlist.yaml` 加载命令白名单（用于 `MiCommander` 终端命令工具的安全审计）。全局配置通过 `conf.GetConfig()` 获取。
- **`pkg/logger/`** — Uber Zap 日志。按级别分文件输出（debug/info/warn/error.log），dev 模式控制台彩色，prod 模式 JSON。
- **`pkg/auth/`** — JWT Token 生成与验证。
- **`pkg/errorer/`** — 统一错误码与错误包装。
- **`pkg/task/`** — 异步任务管理，`task.Do(ctx, fn)` 提供 context 感知的任务执行。
- **`pkg/qdrant/`** — Qdrant gRPC 客户端初始化，用于 RAG 向量存储连接。
- **`pkg/cache/`** — Redis 缓存封装（go-redis/v8/v9）。预留待启用。
- **`pkg/res/`** — Redis 客户端工厂和 HTTP 响应格式工具。
- **`pkg/utils/`** — 通用工具函数（hash、random 等）。
- **`internal/domain/`** — 核心接口定义。`agent.go` 定义 DTO（`TalkReq`, `MemoryReq/Resp`, `RebackReq/Resp` 等），`bridge.go` 定义 `AgentService` 和 `Agent` 接口，实现 service ↔ executor 解耦。
- **`internal/api/routes/`** — Gin 路由注册。完整路由表：
  - `POST /api/ai/chat` — 流式对话（SSE）
  - `GET /api/memory`, `GET /api/memory/:id`, `POST /api/memory/exchange/:id`, `POST /api/memory/clear` — 记忆管理
  - `GET /api/memory/reback`, `POST /api/memory/reback/:index` — 对话回退
  - `GET /api/prompt`, `POST /api/prompt`, `POST /api/prompt/reset` — 系统提示词管理
  - `POST /api/admin/reload` — 配置热重载
- **`internal/api/handler/agenthandler/`** — HTTP Handler，按功能拆分文件（chat/prompt/memory/reback 等）。
- **`internal/api/dto/request/` / `response/`** — 请求/响应 DTO，按模块分子目录（`agentreq/`, `agentresp/`, `adminresp/`）。
- **`internal/api/middlewares/`** — JWT 认证 + CORS 中间件。
- **`internal/service/agentservice/`** — Agent 服务层，实现 `domain.AgentService`，通过 `task.Do` 包装调用 executor。
- **`internal/ai/agent/`** — Eino ADK 多 Agent 编排。6 个子 Agent 加上 Orchestrator，按任务复杂度模型分级：
  - **MiTalker** (haiku) — 日常对话
  - **MiEditer** (sonnet) — 文件读取、写入、创建
  - **MiSummarizer** (sonnet) — 文档摘要 + 知识库工具（注入 `KnowledgeTools`）
  - **MiPlanner** (opus) — 项目计划与方案
  - **MiCommander** (sonnet) — 终端命令执行（受白名单约束）
  - **MiAuditor** (opus) — 代码与配置安全审计
  - **Mifer** (default, Orchestrator) — `deep.New` 编排器，MaxIteration=0（由模型自主控制迭代次数），`EmitInternalEvents: true` 转发子 Agent 事件到 TUI 侧边栏
- **`internal/ai/executor/`** — `adk.Runner` 包装器。`Chat()` 执行 agent 迭代，处理流式/非流式消息，自动追加记忆并保存。`tokens.go` 独立管理 TokenUsage 累计统计。
- **`internal/ai/callback/`** — 全局 Tool 回调处理器。统一处理工具调用事件（开始/结束/错误），替代 executor 内手动处理工具事件的逻辑。
- **`internal/ai/llm/`** — 多后端 ChatModel 管理（Registry 模式）。支持 openai/claude/gemini/ollama 四种 provider，按名称索引（default/haiku/sonnet/opus/multi_modal），缺失后端自动 fallback 到 default。
- **`internal/ai/prompt/`** — 系统提示词管理。`build.go` 构建完整提示词（系统提示词 + 记忆上下文），支持运行时通过 API 动态修改。
- **`internal/ai/memory/`** — JSONL 文件持久化对话历史。dev 模式存 `./memory/{workdir_basename}/{id}.jsonl`，prod 模式存 `~/.mifer/memory/...`。支持列表、加载、切换、清除、回退操作。
- **`internal/ai/tools/`** — 工具定义（Function Calling）。包含：`knowledgesearch`（知识库检索）、`knowledgestore`（文档入库）、`filereader`/`fileviewer`/`filecreator`/`filewriter`（文件操作）、`commandexecutor`（命令执行）、`imagegenerator`（图片生成）。工具通过闭包注入依赖（如 RAG 服务）。
- **`internal/ai/rag/`** — RAG 检索增强。`LazyService` 懒加载模式：`Init()` 仅创建 embedder/loader/chunker（无网络调用），首次工具调用时才通过 `ensureReady()` 连接 Qdrant（Mutex 保护，失败可重试）。子目录：`chunker/`（文档切分）、`embedder/`（Ollama 嵌入）、`loader/`（文件加载，支持 PDF/Word/Text/Markdown）、`vectorstore/`（Qdrant 向量存储封装）。
- **`cmd/bootstrap/`** — 应用启动引导，Application 结构体及初始化方法。
- **`cli/`** — CLI 客户端（Bubble Tea TUI）。通过 HTTP + SSE 调用服务端，核心组件：
  - `cli/tui/` — Bubble Tea 界面（`init.go` 初始化、`update.go` 消息循环、`view.go` 主视图、`stream.go` 流式接收、`sidebar.go` 侧边栏、`command.go` 命令处理、`reback.go` 回退界面、`memory.go` 记忆界面、`system.go` 系统提示词界面）
  - `cli/render/` — 终端渲染（Glamour Markdown 引擎 + Lip Gloss 样式）
  - `cli/client/` — HTTP API 客户端（`chathandler/`, `memhandler/`, `excmemhandler/`, `clearhandler/`, `rebackhandler/`, `prompthandler/`, `reloadhandler/`）

### 关键依赖

- **CloudWeGo Eino** (`github.com/cloudwego/eino v0.8.13`) — ADK、DeepAgent 编排
- **Eino 扩展** — model/openai, model/claude, model/gemini, model/ollama, embedding/ollama, indexer/qdrant, retriever/qdrant, document/loader, document/transformer
- **Gin** (`v1.12.0`) — HTTP 框架
- **Bubble Tea + Bubbles + Glamour + Lip Gloss** — TUI 框架与终端渲染
- **Qdrant** (`github.com/qdrant/go-client v1.15.2`) — RAG 向量存储
- **Uber Zap** — 结构化日志
- **Viper** — 配置管理

## 代码约定

- 所有注释和日志消息使用中文
- `pkg/` 下的包不依赖 `internal/`，可被任意位置导入
- `internal/` 下的包不应被外部项目导入
- 初始化模式：包内 `type` 文件定义结构体，`init.go` 或 `new.go` 提供构造函数，功能拆分到独立文件
- 配置通过 `conf.GetConfig()` 全局获取，不在函数间层层传递
- DTO 按模块分 request/response 子目录
- Handler 按业务模块分组到 `internal/api/handler/` 子目录
- 依赖注入在 `routes/router.go` 的 `NewRouter` 中完成：`executor → service → handler`
- RAG 使用懒加载 + 工具闭包注入，避免阻塞启动和强绑定

## 新增功能指南

### 新增 Agent
1. 在 `internal/ai/agent/` 创建子 Agent 定义（参考 `chatagent.go`、`planner.go` 等），接收 `model.BaseChatModel` 及可选的 `model.ToolCallingChatModel` 参数
2. 在 Orchestrator（`agent/init.go`）的 `deep.New` 配置中注册新子 Agent，通过 `registry.Get("<backend>")` 分配模型
3. 如需新工具，在 `internal/ai/tools/` 定义

### 新增 LLM Provider
1. 在 `internal/ai/llm/providers.go` 定义新的 provider 结构体，实现 `Provider` 接口的 `Name()` 和 `InitModel()` 方法
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
