# Mifer — AI Agent Bot

基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 构建的智能 AI Agent，支持多 Agent 编排、流式对话、工具调用与 CLI / Web 双模交互。

> **一句话定位：** 一个可编程、可扩展的桌面级 AI 助手，既提供开箱即用的终端对话体验，也为开发者提供 Agent 编排与 MCP/Skills 扩展框架。

![TUI 界面截图](./docs/tui-chat.png)

---

## 项目亮点

### 多 Agent 编排与模型路由

基于 CloudWeGo Eino ADK 构建了 5 子 Agent + 1 Orchestrator 的协作体系，按任务类型和复杂度自动路由到不同模型：

| Agent         | 职责               | 模型   |
| ------------- | ------------------ | ------ |
| **MiEditer**  | 文件读写与创建     | sonnet |
| **MiSummarizer** | 文档摘要 + 知识库检索 | sonnet |
| **MiPlanner**  | 项目计划与方案设计 | opus   |
| **MiCommander** | 终端命令执行（白名单约束） | sonnet |
| **MiAuditor**  | 代码与配置安全审计 | opus   |
| **Mifer**      | 编排器，协调子 Agent，由模型自主控制迭代次数 | default |

所有 Agent 通过 `adk.Runner` 统一启动，Agent / ChatModel / Tool 三层接口解耦。

### RAG 检索增强：懒加载 + 工具闭包

知识库检索以**可选工具**形式接入——LLM 在对话中自主判断何时检索、何时入库，不需要预设规则。启动时仅创建 embedder / loader / chunker（零网络调用），首次使用才连接 Qdrant；连接失败不阻塞 Agent，下次调用可重试。向量存储经历了 Milvus → Qdrant 的迁移，最终选型 Qdrant 以降低部署复杂度。

### 对话回退（Reback）

支持将对话回退到历史任意轮次后重新生成。底层在 JSONL 文件中按索引截断，`AgentService.Reback(ctx, index)` 统一接口，同时清理内存中的 Agent 状态，保证回退后对话连续性。

### 配置热重载

`/reload` 命令或 `POST /api/admin/reload` 接口触发，运行时重新加载 YAML 配置和命令白名单，无需重启服务。适用于动态切换模型、调整参数等场景。

### 多模态与工具生态

- **文件查看器**：支持 PDF / Word / Markdown / 纯文本的加载与分块
- **图片生成器**：通过 API 调用图片生成服务
- **知识库工具**：`knowledge_search` 检索 + `knowledge_store` 入库，文档自动切分与向量化

### 系统提示词管理

运行时通过 API 动态读取 / 修改 / 重置系统提示词，修改后立即生效于后续对话。支持多提示词模板管理，与记忆上下文自动拼接。

### 全局工具回调

基于 Eino 全局回调机制统一处理所有工具调用事件（开始 / 结束 / 错误），替代了早期分散在各 executor 中的事件处理代码。TUI 侧边栏通过回调事件实时展示工具执行状态。

### 独立 Token 统计

`tokens.go` 独立管理 TokenUsage 累计统计，与 executor 主逻辑解耦。支持按会话累计、按模型分类，为成本核算提供基础数据。

### MCP 协议支持：外挂式工具生态

基于 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 实现外挂式工具扩展——第三方工具通过 stdio 协议接入，AI 在对话中自动发现和调用，无需修改 Mifer 核心代码。

**架构**：`MCP Manager` 管理所有 Server 连接的生命周期（启动/热重载/关闭）→ `MCPToolAdapter` 将 MCP Tool 的 JSON Schema 自动转换为 Eino `tool.InvokableTool` → `GetToolsForAgent(agentName)` 按 Agent 名路由工具。

**关键设计**：
- **工具适配层** — Schema 通过 JSON 桥接自动转换，无需手工映射；工具名以 `{serverName}_{toolName}` 命名空间隔离
- **Agent 级分配** — 每个 MCP Server 配置 `agents` 字段指定工具分配给哪些子 Agent，空或 `["*"]` 表示全部可用
- **热重载** — `Reload()` 对比新旧配置增量更新（新增/删除/配置变更），不停机
- **失败隔离** — 单个 Server 连接失败不阻塞其他 Server 和 Agent 启动，标记 Status=error 后可重试
- **状态可观测** — `GET /api/mcp/status` 返回所有 Server 的连接状态与工具数量，CLI `/mcp` 命令实时查看
- **进程隔离** — MCP Server 以 stdio 子进程运行，工具调用结果通过适配层过滤，错误不暴露给终端用户

### Skills 技能系统：声明式自定义技能

Skills 允许用户通过 **YAML frontmatter + Markdown 指令** 声明式定义技能，支持 `inline`（内联）和 `fork`（分叉）双模式执行。

**技能示例**：
```markdown
---
name: my-skill
description: 我的自定义技能
context: fork
agent: MiEditer
---

# 技能指令
当此技能被调用时，请按以下步骤操作...
```

**关键设计**：
- **inline 模式** — 技能内容直接注入当前对话上下文，LLM 在同一 Agent 中遵循指令执行
- **fork 模式** — 通过 `AgentHub` 查找目标 Agent，创建子 Agent 独立执行技能指令；目标 Agent 不存在时自动降级为 inline
- **AgentHub 依赖反转** — 技能系统通过 `AgentHub` 接口查找 Agent，不直接依赖 `internal/ai/agent`，Agent 注册由 Orchestrator 启动时完成
- **文件系统即数据库** — 技能以 `目录名/SKILL.md` 形式存储，零配置、零依赖。首次启动自动创建 `hello-world` 示例技能
- **LLM 自主选择** — `skill` 工具的描述中动态注入所有可用技能列表，LLM 根据用户意图自主判断是否调用、调用哪个

### Plan 管理：AI 自主的计划系统

Plan 功能的设计哲学是**"由 AI 决定，而非框架强制"**——不使用 Graph/Workflow 的强制编排，让 LLM 自主调度计划。

**关键设计**：
- **无 Graph 强制** — `MiPlanner` Agent 配备 `PlannerTools()`（仅限文件创建和写入，工作目录锁定在 `.mifer/plans/`），AI 直接编写 Markdown 计划文件，不预设状态机、不限制格式
- **面向 AI 能力演进** — 随着 LLM 推理能力增强，许多需要工程化 Graph 编排的场景可以由 AI 自主完成。**用 AI 的判断替代代码的分支逻辑**，减少工程化复杂度
- **工作目录隔离** — 文件操作限制在 `.mifer/plans/` 下，安全边界在工具层保证
- **CLI 集成** — `/plan` 命令查看计划文件列表，回车加载并展示计划内容

### SSE 流取消

TUI 模式下支持 `Ctrl+C` 中断正在生成的 SSE 流——取消后对话记录保留已生成的部分内容，不会丢失上下文。

---

## 快速开始

### 环境要求

- Go 1.25+

### 安装运行

```bash
git clone <repo-url> mifer
cd mifer
go mod tidy

# 开发模式（默认端口 8080，同时启动 HTTP 服务 + CLI）
go run ./cmd/main

# 仅启动 HTTP 服务
go run ./cmd/main serve

# 仅启动 CLI（需先启动服务）
go run ./cmd/main chat

# 生产模式
MIFER_ENV=prod go run ./cmd/main serve
```

### 配置

首次运行自动生成默认配置文件：

| 模式   | 路径                            |
| ------ | ------------------------------- |
| dev    | `./config/dev.yaml`             |
| prod   | `~/.mifer/config/prod.yaml`     |

关键环境变量（优先级高于配置文件）：

| 变量名                  | 说明                       |
| ----------------------- | -------------------------- |
| `MIFER_AI_BASEURL`      | AI API 地址                |
| `MIFER_AI_APIKEY`       | AI API 密钥                |
| `MIFER_AI_MODEL`        | 默认模型名称               |
| `MIFER_JWT_SECRET`      | JWT 签名密钥               |
| `MIFER_ENV`             | 运行模式（dev / prod）     |

完整的配置项与后端模型管理详见配置文件。

---

## 功能一览

### 对话

- 流式 SSE 响应（`text/event-stream`），实时逐词输出，事件类型区分内容与推理
- 多后端 ChatModel：OpenAI 兼容 / Claude / Gemini / Ollama，缺失的后端自动 fallback
- 模型按能力分级：haiku（轻量对话）、sonnet（文件 / 命令）、opus（计划 / 审计），多模态模型独立配置
- Token 消耗按会话累计统计，persist 到记忆文件

### Agent 编排

- Eino ADK Orchestrator 协调 5 个子 Agent，`MaxIteration=0` 由模型自主控制迭代次数
- `domain.AgentService` 接口隔离 HTTP 层与 AI 核心，方便 mock 与替换实现
- 子 Agent 事件（工具调用、状态变更）通过 `EmitInternalEvents` 转发到 CLI 侧边栏

### 对话记忆

- JSONL 文件持久化，增量追加 + 锁保护并发写入
- 多会话隔离（workdir 哈希 → session ID），支持列表 / 切换 / 清除 / 回退
- 记忆自动追加：用户消息与 AI 回复在 `AgentService.Chat()` 中统一完成

### CLI 客户端

- **TUI 模式**：Bubble Tea 全屏终端，Elm 架构（Model / Update / View），支持鼠标
- **Markdown 渲染**：Glamour 引擎（代码高亮 + 表情符号 + 表格），lipgloss 降级渲染兜底
- **侧边栏**：实时展示当前 Agent、模型、Token 消耗、工具执行状态
- **流式展示**：消息实时追加，推理过程以动画呈现
- **会话管理**：`/viewmemory` 查看历史，`/excmem` 切换会话
- **可配置样式**：主题色、消息样式、滚动指示器、水平滚动宽度均支持自定义

### HTTP API

| 方法   | 路径                       | 说明                   |
| ------ | -------------------------- | ---------------------- |
| POST   | `/api/ai/chat`             | 流式对话（SSE）        |
| GET    | `/api/memory`              | 记忆列表               |
| GET    | `/api/memory/:id`          | 获取指定会话记忆       |
| POST   | `/api/memory/exchange/:id` | 切换记忆会话           |
| POST   | `/api/memory/clear`        | 清除当前记忆           |
| GET    | `/api/memory/reback`       | 获取回退索引列表       |
| POST   | `/api/memory/reback/:index`| 回退到指定轮次         |
| GET    | `/api/prompt`              | 获取系统提示词         |
| POST   | `/api/prompt`              | 修改系统提示词         |
| POST   | `/api/prompt/reset`        | 重置为默认提示词       |
| POST   | `/api/admin/reload`        | 热重载配置与白名单     |

### 工程基础

- **结构化日志** — Uber Zap，按级别分文件（debug/info/warn/error），dev 彩色控制台 / prod JSON
- **JWT 认证** — 中间件已实现，CORS 已配置，路由可选启用
- **错误码体系** — `pkg/errorer` 统一错误码定义与包装
- **异步任务** — `task.Do(ctx, fn)` 统一管理 goroutine 生命周期
- **CI/CD** — GitHub Actions，Tag 推送自动构建 Windows / Linux 多架构二进制

---

## 核心设计决策

### 1. 为什么自建记忆而非依赖框架内置 Memory？

Eino ADK 自带内存记忆，但它绑定于进程生命周期，重启即丢失。Mifer 自建 JSONL 文件记忆层，**增量追加 + 锁保护并发写入**，兼具零依赖的部署便利性和持久性。同时通过 `internal/domain.AgentService` 接口隔离记忆实现，后续可平滑切换为 SQLite 或向量数据库。

### 2. LLM 后端为什么用 Registry 模式？

项目需要同时接入多个模型（日常用 DeepSeek，复杂任务用 Claude，本地测试用 Ollama），且各提供商的 ChatModel 创建方式不同。Registry 模式将模型实例按名称索引（default/haiku/sonnet/opus/multi_modal），业务代码通过 `registry.Get("sonnet")` 获取，**切换模型不改业务代码**。缺失后端自动 fallback 到 default，保证可用性。

### 3. 为什么 Agent 编排设 0 轮迭代？

当前是自主任务执行 Agent。0 轮迭代，由模型自主控制迭代次数，避免无限反思循环和出现迭代次数超出而错误。

### 4. 为什么设计 serve / chat / default 三种启动模式？

Mifer 的核心是一个 HTTP 服务，但通过 `main()` 的参数分发实现了三种启动形态：

```
go run ./cmd/main          → 同时启动服务 + CLI（default）
go run ./cmd/main serve    → 仅启动 HTTP 服务（生产部署）
go run ./cmd/main chat     → 仅启动 CLI 客户端（连接已有服务）
```

架构上，CLI 和服务端之间通过 HTTP + SSE 通信，CLI 本身不直接依赖 `internal/` 的任何模块。这意味着：

- **同一套 HTTP API** 同时服务于 CLI 和未来的 Web UI，不需要两套接口
- **CLI 可独立连接到远程服务**：`chat` 模式下 CLI 仅作为 HTTP 客户端，不加载 LLM 模型、不初始化记忆——所有 AI 能力由远端服务提供
- **default 模式自动编排**：启动服务后 sleep 1s 等待就绪，再启动 CLI，两个组件通过 channel 同步退出，`Ctrl+C` 同时关闭两者

### 5. RAG 为什么用懒加载 + 工具闭包，而非全局注入？

RAG 的 Qdrant 连接需要网络，如果 Agent 初始化时强行连接，不仅拖慢启动速度，还会在没有 Qdrant 的环境中直接报错导致整个 Agent 不可用。Mifer 的方案是将 RAG 的能力通过闭包注入工具，让 AI 在对话中自主调用：

**懒加载层** (`LazyService`)：
```
Init() → NewLazyService()   // 仅创建 embedder / loader / chunker，无网络调用，即时返回
         ↓
首次工具调用 → ensureReady()  // 此时才连接 Qdrant，创建 indexer / retriever
         ↓                 // MuTex 保护，失败后下次调用可重试
         组装为完整 Service
```

**工具闭包注入** (`tools.KnowledgeTools(ragSvc)`)：
```go
func New(ragSvc rag.RAGService) (tool.InvokableTool, error) {
    return utils.InferTool("knowledge_search", "检索知识库...", func(ctx, input) {
        docs, _ := ragSvc.RetrieveWithContext(ctx, query, ctxSize) // 闭包捕获 ragSvc
        return KnowledgeSearchOutput{Results: ragSvc.FormatDocs(docs)}
    })
}
```

这个设计的要点：

1. **RAG 不是框架强制的依赖**，而是 AI 可选的工具——Agent 初始化时 `KnowledgeTools(ragSvc)` 为 nil 时静默返回空工具列表，不影响其他 Agent 正常工作
2. **懒初始化零等待**：启动时 `NewLazyService()` 不触碰网络，启动速度不受 Qdrant 影响；用户不触发知识库功能就永远不连接
3. **失败可恢复**：`ensureReady()` 用 `sync.Mutex` 而非 `sync.Once`，上次连接失败后下次调用可重试，不像 Once 那样失败即永久不可用
4. **AI 自主决策**：工具通过闭包持有 RAG 接口，LLM 在对话中判断何时检索知识库、何时存文档——不需要开发者预设规则

---

## 架构

```
┌─────────────────────────────────────────────┐
│                  cmd / bootstrap             │
│              (入口 + 依赖装配)                 │
├─────────────────────────────────────────────┤
│               internal/api                   │
│    routes → handler → dto (HTTP 层)          │
├─────────────────────────────────────────────┤
│             internal/service                 │
│       AgentService (业务编排 + 记忆管理)       │
├─────────────────────────────────────────────┤
│               internal/ai                    │
│   agent / executor / llm / memory / tool     │
│        (AI 核心，不含 HTTP 依赖)              │
├─────────────────────────────────────────────┤
│                  pkg                         │
│   conf / logger / auth / res / utils ...    │
│          (公共基础包，无业务依赖)               │
└─────────────────────────────────────────────┘
```

分层依赖：`cmd` → `api` → `service` → `ai` → `pkg`，每层只依赖下层，`pkg` 完全不依赖 `internal`。

---

## 项目结构

```
mifer/
├── cli/                       # CLI 客户端
│   ├── client/                #   HTTP API 调用
│   ├── render/                #   终端渲染（Glamour + Lip Gloss）
│   └── tui/                   #   TUI 界面（Bubble Tea）
├── cmd/
│   ├── main/                  #   服务主入口
│   └── bootstrap/             #   启动引导（配置→日志→路由→CLI）
├── config/
│   └── dev.yaml               #   开发环境配置（自动生成）
├── internal/
│   ├── ai/
│   │   ├── agent/             #   Eino ADK 多 Agent 编排
│   │   ├── executor/          #   adk.Runner 包装器
│   │   ├── llm/               #   多后端 ChatModel 管理（Registry）
│   │   ├── memory/            #   JSONL 对话记忆持久化
│   │   └── tool/              #   Function Calling 工具定义
│   ├── api/
│   │   ├── dto/               #   请求 / 响应 DTO
│   │   ├── handler/           #   HTTP Handler（chat / memory）
│   │   ├── middlewares/       #   JWT 认证 + CORS
│   │   └── routes/            #   Gin 路由注册
│   ├── domain/                #   核心接口（AgentService、Agent）
│   └── service/               #   业务编排层
├── pkg/
│   ├── auth/                  #   JWT Token
│   ├── conf/                  #   Viper 配置管理（全局单例）
│   ├── errorer/               #   统一错误码
│   ├── logger/                #   Uber Zap 日志封装
│   ├── res/                   #   统一 HTTP 响应格式
│   └── task/                  #   异步任务管理
├── .github/workflows/         #   CI/CD
└── docs/                      #   截图（请在此添加实际截图）
```

---

## 技术栈

| 组件     | 选型                                  | 说明                        |
| -------- | ------------------------------------- | --------------------------- |
| 语言     | Go 1.25                               |                             |
| HTTP     | Gin v1.12                             | 轻量高性能路由                |
| AI 编排  | CloudWeGo Eino v0.8 (ADK)             | 字节跳动开源 Agent 框架       |
| 默认模型 | DeepSeek V4                           | OpenAI 兼容协议              |
| TUI      | Bubble Tea + Bubbles                  | Elm 架构的终端 UI 框架       |
| 终端渲染 | Glamour + Lip Gloss                   | Markdown + 声明式样式        |
| 日志     | Uber Zap                              | 结构化、高性能                |
| 配置     | Viper                                 | 多源配置 + 环境变量覆盖       |
| 认证     | JWT                                   | 无状态 Token 认证            |
| CI/CD    | GitHub Actions                        | Tag 触发多架构构建            |

---

## 后续方向

- MCP Server 模式——让 Mifer 自身作为 MCP Server 对外暴露能力
- RAG 语义索引增强——本地代码库级语义检索
- Web UI 管理面板
- Docker 一键部署
- 会话分支与多路线对话探索
