# Mifer — AI Agent Bot

基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 构建的智能 AI Agent，支持多 Agent 编排、MCP 协议、技能系统、RAG 检索增强与 CLI/Web 双模交互。

---

## 操作指南

### 环境要求

- Go 1.21+
- Redis（可选，缓存加速）

### 快速开始

```bash
# 克隆项目
git clone <repo-url> mifer
cd mifer

# 安装依赖
go mod tidy

# 开发模式运行（端口 8080，控制台彩色日志）
go run ./cmd/main

# 生产模式运行（JSON 日志，配置存 ~/.mifer/）
MIFER_ENV=prod go run ./cmd/main
```

### 配置

首次运行自动生成默认配置文件：

| 模式 | 配置文件路径 |
|------|-------------|
| dev | `./config/dev.yaml` |
| prod | `~/.mifer/config/prod.yaml` |

关键环境变量（优先级高于配置文件）：

| 变量名 | 说明 |
|--------|------|
| `MIFER_AI_BASEURL` | AI API 地址（默认 DeepSeek） |
| `MIFER_AI_APIKEY` | AI API 密钥 |
| `MIFER_AI_MODEL` | 模型名称（默认 deepseek-v4-flash） |
| `MIFER_ENV` | 运行模式（dev / prod） |

### CLI 使用

```bash
# 编译 CLI
go build -o mifer-cli ./cli

# 交互式对话
./mifer-cli chat

# 单次问答
./mifer-cli ask "用 Go 实现快速排序"

# 管道模式
cat error.log | ./mifer-cli ask "分析这段日志"
```

### API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/ai/chat` | 对话接口（流式 SSE） |
| GET | `/api/memory/{id}` | 获取对话历史 |
| DELETE | `/api/memory/{id}` | 清除对话记忆 |

---

## 功能规划目标

### AI 核心

- [x] LLM 对话（OpenAI 兼容协议，支持 DeepSeek / OpenAI / Claude 等）
- [x] 多 Agent 编排（ADK Orchestrator + 子 Agent，最大 3 轮迭代）
- [x] 流式响应（SSE 实时逐词输出）
- [x] 对话记忆（JSON 文件持久化，多会话隔离，自动分片）
- [x] 工具调用（Function Calling / Tool Use）
- [ ] 多轮任务拆解与执行（Plan → Execute → Verify）
- [ ] 自主反思与纠错（Self-Reflection Loop）
- [ ] 人机协作回退（不确定时主动询问用户）

### MCP 协议

- [ ] MCP Client（连接外部 MCP Server，接入第三方工具）
- [ ] MCP Server（将 Mifer 能力暴露为 MCP 服务，供其他 Agent 调用）
- [ ] 内置工具集
  - [ ] 文件系统操作（读写、搜索、替换）
  - [ ] Shell 命令执行（沙箱安全策略）
  - [ ] HTTP 请求代理
  - [ ] 数据库查询（MySQL / PostgreSQL / SQLite）
  - [ ] Git 操作（log、diff、blame）
- [ ] MCP 工具市场（社区共享工具集）

### Skills 技能系统

- [ ] 技能注册与生命周期管理
- [ ] 技能热加载（配置变更后无需重启）
- [ ] 内置技能
  - [ ] 代码审查（Code Review，输出建议与风险点）
  - [ ] 文档生成（从代码自动生成 API 文档）
  - [ ] 测试生成（单元测试 / 表驱动测试）
  - [ ] 错误分析（从堆栈追踪定位根因）
  - [ ] SQL 优化（慢查询分析与改写建议）
- [ ] 用户自定义技能
  - [ ] Lua 脚本扩展（轻量沙箱运行）
  - [ ] YAML 声明式技能（Prompt + Tool 组合）
- [ ] 技能管道（Skill Pipeline，多个技能链式调用）

### Rules 规则引擎

- [ ] 系统级规则（安全边界、敏感信息过滤、输出长度限制）
- [ ] 用户级规则
  - [ ] 回答风格偏好（简洁 / 详细 / 学术化）
  - [ ] 语言偏好（中文 / 英文 / 双语）
  - [ ] 角色设定（技术专家 / 导师 / 助手）
- [ ] 项目级规则
  - [ ] 从 `.mifer/rules/` 目录自动加载
  - [ ] 代码规范约束（命名、架构、最佳实践）
  - [ ] 项目特定知识（技术栈、业务术语）
- [ ] 规则优先级与冲突解决策略
- [ ] 规则模板商店

### RAG 检索增强

- [ ] 本地文件索引
  - [ ] 代码库语义索引（理解项目结构）
  - [ ] Markdown / PDF / 纯文本文档索引
  - [ ] 增量更新（文件变更自动同步）
- [ ] 向量数据库集成
  - [ ] Milvus（生产级大规模检索）
  - [ ] Chroma（轻量本地部署）
  - [ ] 内存向量库（零依赖快速启动）
- [ ] 智能上下文注入
  - [ ] 查询意图识别与检索策略选择
  - [ ] 多路召回（关键词 + 向量 + 图谱）
  - [ ] 相关性排序与去噪
- [ ] 知识库管理
  - [ ] 多知识库隔离与切换
  - [ ] 知识库导入导出
  - [ ] 知识新鲜度追踪

### CLI 交互

- [x] 基础命令行框架
- [ ] 交互式 REPL 模式
  - [ ] 多行输入（代码块粘贴）
  - [ ] 命令补全（Tab Completion）
  - [ ] 历史搜索（Ctrl+R）
  - [ ] 快捷键绑定（Ctrl+C 中断、Ctrl+D 退出）
- [ ] 终端渲染增强
  - [ ] Markdown 渲染（标题、列表、表格、代码块语法高亮）
  - [ ] 流式输出逐词打印
  - [ ] 进度条与状态提示
  - [ ] 自适应终端宽度
- [ ] 会话管理
  - [ ] 多会话创建与切换（`/session switch`）
  - [ ] 会话列表与搜索（`/session list`）
  - [ ] 对话导出（JSON / Markdown / HTML）
  - [ ] 上下文压缩与摘要
- [ ] 上下文感知
  - [ ] 自动加载当前工作目录为上下文
  - [ ] Git 变更感知（分支、diff、log）
  - [ ] 终端输出捕获与分析

### Web UI

- [ ] 对话界面（类 ChatGPT 布局）
  - [ ] 流式消息展示
  - [ ] Markdown + 代码高亮渲染
  - [ ] 工具调用过程可视化
- [ ] 会话管理面板（列表、搜索、删除、导出）
- [ ] 配置管理面板（模型切换、参数调整、规则编辑）
- [ ] 知识库管理界面（上传文件、索引状态、检索测试）
- [ ] 系统监控面板（Token 用量、响应延迟、错误率）

### 记忆与上下文

- [x] 短期记忆（对话窗口滑动、Token 感知截断）
- [ ] 长期记忆
  - [ ] 用户画像（角色、偏好、知识背景）
  - [ ] 事实记忆（用户告知的重要信息）
  - [ ] 经验记忆（历史任务与解决方案）
- [ ] 工作记忆（当前任务拆解与中间状态）
- [ ] 记忆管理
  - [ ] 记忆重要性评分与淘汰
  - [ ] 冗余记忆合并
  - [ ] 遗忘策略（LRU / 时间衰减 / 语义相似度去重）
- [ ] 记忆存储后端
  - [ ] JSON 文件（单机零依赖）
  - [ ] SQLite（结构化检索）
  - [ ] Redis（高性能缓存层）
  - [ ] 向量数据库（语义检索）

### 多模态

- [ ] 图片理解（视觉问答、截图分析、图表解读）
- [ ] 图片生成（通过 DALL-E / Stable Diffusion API）
- [ ] 语音输入（ASR 转文本）
- [ ] 语音输出（TTS 朗读回复）
- [ ] 文件解析（Office 文档、PDF、EPUB）

### 集成与扩展

- [ ] 多模型路由
  - [ ] 按任务复杂度自动选模型（简单→轻量模型，复杂→强模型）
  - [ ] 故障转移（主模型不可用时自动切换备用）
  - [ ] 本地模型支持（Ollama / vLLM）
- [ ] Webhook 触发器（GitHub Event → Mifer 自动响应）
- [ ] 定时任务（Cron Agent，定时检查、日报生成、巡检）
- [ ] 通知集成
  - [ ] 企业微信 / 飞书 / 钉钉
  - [ ] 邮件
  - [ ] Slack / Discord
- [ ] 插件市场
  - [ ] 插件发现与安装
  - [ ] 版本管理与依赖检查
  - [ ] 社区插件评分

### 工程化

- [ ] 测试体系
  - [ ] 单元测试（覆盖率 > 80%）
  - [ ] 集成测试（API 端到端）
  - [ ] 回归测试（核心场景快照对比）
- [ ] Docker 容器化部署
- [ ] CI/CD 流水线
- [ ] 可观测性
  - [ ] 结构化日志（Zap JSON）
  - [ ] Prometheus 指标（请求量、延迟、Token 消耗、错误率）
  - [ ] 分布式追踪（OpenTelemetry）
- [ ] 安全
  - [ ] JWT 认证与鉴权
  - [ ] API 限流（令牌桶 / 滑动窗口）
  - [ ] 敏感信息脱敏
  - [ ] 沙箱执行隔离
  - [ ] Prompt 注入防护

---

## 项目结构

```
mifer/
├── cli/                          # CLI 入口
├── cmd/
│   ├── main/                     # 服务主入口（main.go）
│   └── bootstrap/                # 启动引导（配置、日志、路由初始化）
├── config/
│   └── dev.yaml                  # 开发环境配置文件（自动生成）
├── internal/
│   ├── ai/
│   │   ├── agent/                # Eino ADK 多 Agent 编排
│   │   ├── executor/             # adk.Runner 包装器
│   │   ├── llm/                  # OpenAI 兼容 ChatModel 初始化
│   │   ├── memory/               # 对话历史持久化（JSON 文件）
│   │   └── tool/                 # 工具定义（Function Calling）
│   ├── api/
│   │   ├── dto/
│   │   │   ├── request/          # 请求 DTO
│   │   │   └── response/         # 响应 DTO
│   │   ├── handler/
│   │   │   └── agenthandler/     # HTTP Handler（chat/memory）
│   │   ├── middlewares/          # JWT 认证、CORS 中间件
│   │   └── routes/              # Gin 路由注册
│   ├── domain/                   # 核心接口与 DTO 定义
│   └── service/
│       └── agentservice/         # Agent 服务层（业务编排）
├── pkg/
│   ├── auth/                     # JWT Token 生成与验证
│   ├── cache/                    # Redis 缓存封装
│   ├── conf/                     # Viper 配置管理（全局单例）
│   ├── errorer/                  # 统一错误码定义
│   ├── logger/                   # Uber Zap 日志封装
│   ├── res/                      # 统一 HTTP 响应格式
│   ├── task/                     # 异步任务管理
│   └── utils/                    # 通用工具函数
├── CLAUDE.md                     # Claude Code 协作指引
└── README.md
```

### 分层架构

```
cmd → internal/api → internal/service → internal/ai → pkg
```

- **`cmd`** — 应用入口与启动引导
- **`internal/api`** — HTTP 层，路由、中间件、Handler
- **`internal/service`** — 业务编排层
- **`internal/ai`** — AI 核心（Agent、LLM、Memory、Tool）
- **`pkg`** — 公共基础包，无业务依赖

### 技术栈

| 组件 | 技术选型 |
|------|----------|
| 语言 | Go 1.21+ |
| HTTP 框架 | Gin |
| AI 编排 | CloudWeGo Eino (ADK) |
| 默认模型 | DeepSeek V4 (OpenAI 兼容协议) |
| 日志 | Uber Zap |
| 缓存 | Redis (go-redis/v8) |
| 配置 | Viper |
| 认证 | JWT |

---

— 蓝山最终考核
