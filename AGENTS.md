# AGENTS.md — Mifer

> 此文件为 OpenCode 等 AI 工具提供高信号指引。完整架构见 `CLAUDE.md`。

## 构建与运行

```bash
go mod tidy
.\build.bat                       # 一键构建 mifer.exe
go build -o mifer.exe ./cmd/main  # 手动构建（必须 -o，否则生成 main.exe）
go run ./cmd/main                 # 默认：启动服务(端口15555) + CLI
go run ./cmd/main serve           # 仅 HTTP 服务
go run ./cmd/main chat            # 仅 CLI（需先启动服务）
go run ./cmd/main chat --<id>     # 指定会话 ID
MIFER_ENV=prod go run ./cmd/main  # 生产模式
```

- **无 Makefile，无 lint 配置（如 golangci-lint），无编辑器配置。**
- Go 1.25.4，Eino v0.8.13。
- 项目无 CI lint/typecheck 步骤 — 构建即验证。

## 外部依赖

通过 `docker-compose.yml` 管理，非必须但影响功能：

| 服务 | 端口 | 用途 |
|------|------|------|
| Qdrant | 6333 (HTTP) / 6334 (gRPC) | RAG 向量存储 |
| SearXNG | 18080 | web_search 后端 |
| Ollama | 11434 | 嵌入模型 (`nomic-embed-text`) |

首次启动后拉取嵌入模型：`docker-compose exec ollama ollama pull nomic-embed-text`

## 测试

**无统一测试命令。** 仅有 7 个零散的 `_test.go` 文件：
`pkg/utils/`, `pkg/errorer/`, `pkg/task/`, `pkg/auth/`, `pkg/exc/`, `internal/ai/memory/`, `internal/ai/prompt/`

运行单个：`go test ./pkg/errorer/`

## 架构要诀

```
cmd → api → service → ai → pkg
CLI (cli/) 通过 HTTP+SSE 与服务端通信，不依赖 internal/
```

- 入口 `cmd/main/main.go`，引导在 `cmd/bootstrap/`
- 核心接口在 `internal/domain/`（`AgentService`、`Agent`、`ToolService`），实现者分别放在 `internal/service/` 和 `internal/ai/executor/`
- 所有可复用错误字符串定义在 `pkg/errorer/errorer.go`，三个构造函数：`New`、`NewS`（wrapping）、`NewF`（格式化）
- 全局配置通过 `conf.GetConfig()` 获取，不层层传递

## 命名与风格

- **包名**：全小写单单词（`agenthandler`、`agentservice`），多单词文件名用 `snake_case`
- **类型文件**：`type.go`；**构造函数**：`init.go` 或 `new.go`
- **注释用中文**，Go doc 格式 `// TypeName 描述`
- **未导出结构体字段** camelCase，导出字段 PascalCase
- **错误常量** `Err` 前缀，API 路径常量 `API` 前缀

## 导入排序

```go
import (
    "context"        // 标准库
    "errors"

    "github.com/cloudwego/eino/schema"  // 第三方

    "mifer/internal/domain"  // 内部包
)
```

## 特殊约定

- **环境不能运行 Linux 命令** — 此项目在 Windows 开发，CI 在 Linux ubuntu-latest 上构建
- **配置自动生成** — 首次运行自动创建 `./config/dev.yaml` 或 `~/.mifer/config/prod.yaml`，无需手动编写
- **端口回退** — 默认 15555，端口不足时自动 +10 递增到 18000
- **工具确认** 默认关闭，需配置 `confirm.enabled: true`
- **不使用 panic 做控制流**，`context.Canceled` 视为正常条件静默返回
- **已知特殊命名**（非拼写错误）：`Humen`、`Prompty`、`Clier`、`Mifer`、`excmem`
- **所有修改遵循现有代码风格**，不自行引入新框架或命名约定
