# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Run

```bash
go build ./cmd/main          # 编译
go run ./cmd/main            # 运行（默认 dev 模式，端口 8080）
MIFER_ENV=prod go run ./cmd/main  # 生产模式运行
```

No Makefile or build scripts — just standard Go tooling.

## Architecture

分层架构，依赖方向：`cmd` → `internal/api` → `internal/service` → `internal/ai` → `pkg`

### 启动流程

`cmd/main/main.go` → `bootstrap.NewApplication()` → `loadConfig` → `initLogger` → `initRouter` → `Run()`

### 各层职责

- **`pkg/conf/`** — Viper 配置管理。`LoadConfig()` 根据 `MIFER_ENV` 加载 `./config/dev.yaml` 或 `~/.mifer/config/prod.yaml`，首次运行自动创建默认配置文件。环境变量可覆盖关键字段（`MIFER_AI_BASEURL`, `MIFER_AI_APIKEY` 等）。全局配置通过 `conf.GetConfig()` 获取。
- **`pkg/logger/`** — Uber Zap 日志。按级别分文件输出（debug/info/warn/error.log），dev 模式控制台彩色输出，prod 模式 JSON 输出。快捷方法：`logger.Info()`, `logger.Error()` 等。
- **`internal/api/routes/`** — Gin 路由。`/api/ai/chat` (POST) 已定义但 handler 尚未绑定。
- **`internal/api/middlewares/`** — JWT 认证中间件（从 token 提取 `user_id`/`user_name` 写入 context）、CORS 中间件。
- **`internal/domain/`** — 核心接口和 DTO。`TalkReq/Resp`、`MemoryReq/Resp`。
- **`internal/ai/agent/`** — Eino ADK 多 Agent 编排。"Mifer" 主 Agent（`deep.New`，Orchestrator）管理 "MiTalker" 子 Agent（`ChatModelAgent`，闲聊）。最大迭代 3 次。
- **`internal/ai/llm/`** — OpenAI 兼容 ChatModel 初始化。默认指向 DeepSeek API（`deepseek-v4-flash`）。
- **`internal/ai/executor/`** — `adk.Runner` 包装器，`Chat()` 执行 agent 并返回输出。
- **`internal/ai/memory/`** — JSON 文件持久化对话历史。dev 模式存 `./memory/{workdir_basename}/{id}.json`，prod 模式存 `~/.mifer/memory/...`。

### 关键依赖

- **CloudWeGo Eino** (`github.com/cloudwego/eino`) — 字节跳动开源的 Go AI 编排框架，提供 ADK、ChatModel、DeepAgent 等
- **Gin** — HTTP 框架
- **Redis** (`go-redis/v8`) — 缓存（已集成但当前路由初始化中未使用）

## 代码约定

- 所有注释和日志消息使用中文
- `pkg/` 下的包不依赖 `internal/`，可被任意位置导入
- `internal/` 下的包不应被外部项目导入
- 初始化模式：包内 `type` 文件定义结构体，`init.go` 或 `new.go` 提供构造函数，功能拆分到独立文件
- 配置通过 `conf.GetConfig()` 全局获取，不在函数间层层传递
