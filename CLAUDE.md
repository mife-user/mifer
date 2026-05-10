# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Mifer — AI Agent Bot，蓝山最终考核项目。基于 CloudWeGo Eino 构建，支持多 Agent 编排、MCP 协议、技能系统、RAG 检索增强与 CLI/Web 双模交互。

## Build / Run

```bash
go build ./cmd/main              # 编译 HTTP 服务
go run ./cmd/main                # 运行（默认同时启动服务+CLI，端口 8080）
go run ./cmd/main serve          # 仅启动 HTTP 服务
go run ./cmd/main chat           # 仅启动 CLI（需先启动服务）
go build -o mifer-cli ./cli      # 编译 CLI 客户端
MIFER_ENV=prod go run ./cmd/main # 生产模式
```

No Makefile or build scripts — just standard Go tooling. Go 1.25.4, Eino v0.7.13.

## Architecture

分层架构，依赖方向：`cmd` → `internal/api` → `internal/service` → `internal/ai` → `pkg`

### 启动流程

`cmd/main/main.go` → `bootstrap.NewApplication()`:
1. `loadConfig` — Viper 加载 YAML 配置，首次运行自动生成默认配置文件
2. `initContext` — 创建带 session ID 的 context（基于 workdir 哈希，用于记忆隔离）
3. `initLogger` — 初始化 Zap 日志
4. `initRouter` — 初始化 Gin 路由（内部创建 `executor → agentservice → agenthandler` 依赖链）
5. `initCli` — 初始化 CLI 客户端（连接到 HTTP 服务）

`main()` 按 `os.Args[1]` 分发：`serve` 仅启动服务，`chat` 仅启动 CLI，默认同时启动两者。

### 各层职责

- **`pkg/conf/`** — Viper 配置管理。`LoadConfig()` 根据 `MIFER_ENV` 加载 `./config/dev.yaml` 或 `~/.mifer/config/prod.yaml`，首次运行通过 `newDefaultCfg()` 自动创建默认配置文件。环境变量可覆盖关键字段（`MIFER_AI_BASEURL`, `MIFER_AI_APIKEY`, `MIFER_AI_MODEL`, `MIFER_REDIS_HOST`, `MIFER_REDIS_PASSWORD`, `MIFER_JWT_SECRET`）。`StatusConfig()` 校验必填项。全局配置通过 `conf.GetConfig()` 获取。
- **`pkg/logger/`** — Uber Zap 日志。按级别分文件输出（debug/info/warn/error.log），dev 模式控制台彩色输出，prod 模式 JSON 输出。快捷方法：`logger.Info()`, `logger.Error()` 等。
- **`pkg/auth/`** — JWT Token 生成与验证。
- **`pkg/cache/`** — Redis 缓存封装（go-redis/v8）。当前 `initRedis` 已注释，预留待启用。
- **`pkg/errorer/`** — 统一错误码与错误包装。
- **`pkg/res/`** — 统一 HTTP 响应格式（`Success()`, `Error()`, `Stream()`）。
- **`pkg/task/`** — 异步任务管理，`task.Do(ctx, fn)` 提供 context 感知的任务执行。
- **`pkg/utils/`** — 通用工具函数（hash、random 等）。
- **`internal/domain/`** — 核心接口定义。`AgentService` 接口（service 层实现）和 `Agent` 接口（executor 实现），实现 service ↔ executor 的解耦。
- **`internal/api/routes/`** — Gin 路由注册。`/api/ai/chat` (POST SSE 流式)、`/api/memory/:id` (GET/DELETE)。
- **`internal/api/handler/agenthandler/`** — HTTP Handler，处理 chat 与 memory 请求。
- **`internal/api/dto/request/` / `response/`** — 请求/响应 DTO。
- **`internal/api/middlewares/`** — JWT 认证中间件（从 token 提取 `user_id`/`user_name` 写入 context）、CORS 中间件。
- **`internal/service/agentservice/`** — Agent 服务层，实现 `domain.AgentService`，通过 `task.Do` 包装调用 executor。
- **`internal/ai/agent/`** — Eino ADK 多 Agent 编排。`Humen` 结构体聚合 `adk.Agent` 和 `*memory.Memory`。"Mifer" 主 Agent（`deep.New`，Orchestrator）管理 "MiTalker" 子 Agent（`ChatModelAgent`，闲聊）。最大迭代 3 次。
- **`internal/ai/executor/`** — `adk.Runner` 包装器。`Chat()` 执行 agent 迭代，处理流式/非流式消息，自动追加记忆并保存。
- **`internal/ai/llm/`** — OpenAI 兼容 ChatModel 初始化（Eino `openai.ChatModel`）。默认指向 DeepSeek API（`deepseek-v4-flash`），支持通过配置切换模型。
- **`internal/ai/memory/`** — JSONL 文件持久化对话历史（每行一条 JSON）。dev 模式存 `./memory/{workdir_basename}/{id}.jsonl`，prod 模式存 `~/.mifer/memory/...`。`AppendUser`/`AppendAssistant` 加锁追加到内存，`Save()` 增量写入文件。
- **`internal/ai/tool/`** — 工具定义（Function Calling）。注册给 Agent 使用的工具集合。
- **`cmd/bootstrap/`** — 应用启动引导，Application 结构体及初始化方法。
- **`cli/`** — CLI 客户端，通过 HTTP 调用服务端 `/api/ai/chat` 接口（SSE 流式），提供 REPL 交互。

### 关键依赖

- **CloudWeGo Eino** (`github.com/cloudwego/eino v0.7.13`) — ADK、ChatModel、DeepAgent 编排
- **Eino OpenAI 扩展** (`github.com/cloudwego/eino-ext/components/model/openai`) — OpenAI 兼容 ChatModel
- **Gin** (`v1.12.0`) — HTTP 框架
- **Redis** (`go-redis/v8`) — 缓存（当前未激活）
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

## 当前实现状态

### 已完成
- [x] LLM 对话（OpenAI 兼容协议，DeepSeek 默认）
- [x] 多 Agent 编排（ADK Orchestrator + 子 Agent，最大 3 轮迭代）
- [x] 流式响应（SSE）
- [x] 对话记忆（JSONL 文件持久化，增量追加，基于 context session ID 隔离）
- [x] 工具调用（Function Calling）
- [x] CLI 客户端（HTTP + SSE，基础 REPL）
- [x] JWT 认证（中间件已实现，路由未强制启用）
- [x] 结构化日志
- [x] 配置管理（dev/prod 双模式，环境变量覆盖）

### 规划中
- [ ] MCP 协议支持（Client / Server）
- [ ] Skills 技能系统（内置技能 + 自定义脚本）
- [ ] Rules 规则引擎（系统/用户/项目三级）
- [ ] RAG 检索增强（向量检索 + 知识库）
- [ ] CLI REPL 完整交互（命令补全、历史搜索、Markdown 渲染）
- [ ] Web UI
- [ ] Redis 缓存激活
- [ ] 长期记忆与用户画像
- [ ] 多模态（图片/语音）
- [ ] 插件市场
- [ ] 可观测性（Prometheus + Tracing）
- [ ] Docker 部署

## 新增功能指南

### 新增 Agent
1. 在 `internal/ai/agent/` 创建子 Agent 定义
2. 在 Orchestrator（`agent/init.go`）的 `deep.New` 配置中注册新子 Agent
3. 如需新工具，在 `internal/ai/tool/` 定义

### 新增 HTTP 接口
1. 在 `internal/api/dto/request/` 和 `response/` 定义 DTO
2. 在 `internal/api/handler/` 对应子目录添加 Handler
3. 在 `internal/api/routes/router.go` 注册路由
4. 业务逻辑放在 `internal/service/` 对应服务层

### 新增 CLI 命令
1. 在 `cli/` 定义命令结构
2. 复用 `internal/service/` 的业务逻辑（通过 HTTP API 调用）

### 新增配置项
1. 在 `pkg/conf/new.go` 的 `defaultConfig` 常量中添加默认值
2. 在 `pkg/conf/type.go` 的结构体中添加字段（带 `mapstructure` tag）
3. 如需环境变量覆盖，在 `pkg/conf/load.go` 的 `applyEnvOverrides` 中添加映射

---

— 蓝山最终考核
