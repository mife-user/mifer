---
name: eino-reference
description: Use when working with CloudWeGo Eino framework (v0.8.13) — creating agents, tools, RAG pipelines, streaming, multi-agent orchestration, or model initialization in this project. Covers adk, deep, schema, model, tool, compose, eino-ext, and RAG APIs.
---

# Eino 框架参考

基于 `github.com/cloudwego/eino v0.8.13`，本项目的核心 AI 框架。

## 核心架构概念

```
用户输入 → Runner.Run(messages) → Agent(Orchestrator) → 子 Agent → Tool → 返回
              ↓                        ↓                    ↓
         事件流迭代器              deep.New 编排         NewChatModelAgent
```

**三层抽象：**
- **Model** — 直接调用 LLM，输入 `[]*Message`，输出 `*Message`
- **Agent** — "模型 + 工具 + ReAct 循环" 的打包，自动处理思考→调工具→看结果
- **Runner** — Agent 执行器，管理对话生命周期和历史

## 快速参考：常见操作

### 创建模型连接

```go
// OpenAI / 兼容接口 (DeepSeek, 通义千问, vLLM)
model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    Model:   "gpt-4o",              // 或 "deepseek-chat"
    BaseURL: "https://api.openai.com/v1", // 兼容接口时改此值
    APIKey:  os.Getenv("OPENAI_API_KEY"),
})

// Claude (Anthropic)
model, _ := claude.NewChatModel(ctx, &claude.Config{
    Model: "claude-sonnet-4-6", APIKey: "...", MaxTokens: 4096,
})

// Ollama (本地)
model, _ := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
    Model: "llama3.1", BaseURL: "http://localhost:11434",
})
```

### 定义工具 (Tool)

**方式一：手动实现接口（最灵活）**
```go
type MyTool struct{}
func (t *MyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "my_tool",
        Desc: "工具描述——模型据此判断何时使用",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "param1": {Type: schema.String, Desc: "参数说明", Required: true},
        }),
    }, nil
}
func (t *MyTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
    // args 是 JSON 字符串: {"param1": "value"}
    // 返回执行结果的纯文本
    return "result", nil
}
```

**方式二：`utils.InferTool`（推荐，自动推断 Schema）**
```go
type MyInput struct {
    FilePath string `json:"file_path" jsonschema:"required,description=文件路径"`
}
type MyOutput struct {
    Content string `json:"content"`
}
func myFunc(ctx context.Context, input MyInput) (MyOutput, error) { ... }

tool, _ := utils.InferTool("tool_name", "工具描述", myFunc)
```

**关键规则：**
- `fn` 签名必须严格为 `func(context.Context, Input) (Output, error)`
- `jsonschema` tag 决定 LLM 看到的参数 Schema —— 务必写 `required` 和 `description`
- 外部依赖通过闭包注入：`func New(cfg *Config) (tool.InvokableTool, error) { ... }`
- Tool 返回 `(string, error)`，框架自动包装为 `ToolMessage`

### 创建 Agent

```go
// 单 Agent
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:         "AgentName",        // 必须唯一
    Description:  "简短描述",
    Instruction:  "系统提示词...",
    Model:        chatModel,          // model.BaseChatModel
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{tool1, tool2},
        },
        EmitInternalEvents: true,     // 子 Agent 事件透传到 UI
    },
    MaxIterations: 20,                // 防止无限循环
})

// 多 Agent 编排器
orchestrator, _ := deep.New(ctx, &deep.Config{
    Name:        "Orchestrator",
    Instruction: "调度策略提示词...",
    ChatModel:   registry.Get("default"),
    SubAgents:   []adk.Agent{agent1, agent2, agent3},
    MaxIteration: 5,
    ToolsConfig: adk.ToolsConfig{EmitInternalEvents: true},
})
```

### 运行 Agent

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
})

// 方式一：runner.Run(messages) — 传入完整消息历史
iter := runner.Run(ctx, messages)
for {
    event, ok := iter.Next()
    if !ok { break }
    if event.Err != nil { return event.Err }
    // 处理 event...
}

// 方式二：runner.Query(input) — Runner 管理历史
iter := runner.Query(ctx, "用户输入")
```

### 事件流消费模式

```go
for {
    event, ok := iter.Next()
    if !ok { break }

    // 检测 Agent 切换（多 Agent 场景）
    if event.AgentName != "" { /* 当前 Agent */ }

    msgOutput := event.Output.MessageOutput
    if msgOutput == nil { continue }

    if !msgOutput.IsStreaming {
        // 非流式：角色判断
        switch msgOutput.Role {
        case schema.Assistant: /* 助手回复 */
        case schema.Tool:      /* 工具结果 */ 
        }
    } else {
        // 流式：逐块读取
        for {
            chunk, err := msgOutput.MessageStream.Recv()
            if errors.Is(err, io.EOF) { break }
            if chunk.ReasoningContent != "" { /* 推理内容 */ }
            fmt.Print(chunk.Content) // 逐字输出
        }
    }
}
```

### Message 操作

```go
// 四种消息角色
schema.SystemMessage("你是AI助手")                        // system
schema.UserMessage("用户问题")                            // user
schema.AssistantMessage("回复内容", toolCalls)             // assistant
schema.ToolMessage("工具执行结果", "tool_name")            // tool

// 检查工具调用
if len(msg.ToolCalls) > 0 {
    name := msg.ToolCalls[0].Function.Name   // 工具名
    args := msg.ToolCalls[0].Function.Arguments // JSON 参数
}
```

### LLM 多后端注册

```go
type Provider interface {
    Name() string                           // "openai" / "claude" / "gemini" / "ollama"
    InitModel(ctx context.Context, cfg *conf.AIConfig) (model.BaseChatModel, error)
}

type Registry struct {
    models   map[string]model.BaseChatModel // 按名称索引
    fallback string                         // 缺失时回退
}
registry.Get("sonnet")  // 缺失时 fallback 到 "default"
```

## RAG 流水线

```
Loader → Transformer → Embedder → Indexer → [VectorStore] → Retriever → ChatTemplate → ChatModel
```

| 组件 | 包路径 | 核心方法 |
|------|--------|----------|
| `Loader` | `document.Loader` | `Load(ctx, src) ([]*schema.Document, error)` |
| `Transformer` | `document.Transformer` | `Transform(ctx, docs) ([]*schema.Document, error)` |
| `Embedder` | `embedding.Embedder` | `EmbedStrings(ctx, texts) ([][]float64, error)` |
| `Indexer` | `indexer.Indexer` | `Store(ctx, docs) ([]string, error)` |
| `Retriever` | `retriever.Retriever` | `Retrieve(ctx, query) ([]*schema.Document, error)` |
| `ChatTemplate` | `prompt.ChatTemplate` | `Format(ctx, msgs) ([]*schema.Message, error)` |

**懒加载模式（本项目采用）：** `LazyService.Init()` 仅创建组件，首次工具调用时才 `ensureReady()` 连接 Qdrant。

## 常见错误与注意事项

| 问题 | 正确做法 |
|------|----------|
| Agent Name 重复 | 同一编排器下 Name 必须唯一 |
| 忘记消费事件流 | 必须消费到 `iter.Next()` 返回 `false`，否则资源泄漏 |
| `ToolCalls` 非空但 Content 非空 | 工具调用请求时 Content 为空字符串 |
| `MaxIterations` 不设或过大 | 建议 3~10（单 Agent）、5~20（编排器），防止无限循环 |
| `fn` 签名不对 | `InferTool` 要求严格 `func(context.Context, Input) (Output, error)` |
| 忽略 `ReasoningContent` | Claude/DeepSeek R1 等思考模型的推理内容在 `chunk.ReasoningContent` |
| 直接调用 `model.Generate` 无工具循环 | 手动调用需自己写 ReAct 循环；用 Agent 则自动处理 |

## 本项目的子 Agent 体系

| Agent | 模型 | 职责 |
|-------|------|------|
| **MiEditer** | sonnet | 文件读取、写入、创建 |
| **MiSummarizer** | sonnet | 文档摘要 + 知识库检索 |
| **MiPlanner** | opus | 项目计划与方案 |
| **MiCommander** | sonnet | 终端命令执行（白名单约束） |
| **MiAuditor** | opus | 代码与配置安全审计 |
| **Mifer** (Orchestrator) | default | 任务调度与路由 |

## ReAct 循环原理

```
1. 模型收到消息 → 返回 ToolCall 或文字
2. 如果是 ToolCall → 执行工具 → 结果发回模型 → 回到步骤 1
3. 如果是文字 → 输出给用户 → 结束
```

Agent (`adk.NewChatModelAgent`) 自动执行此循环，`MaxIterations` 限制最大轮次。

## 回调机制

全局回调通过 `internal/ai/callback/` 处理 Tool 事件：
- `OnStart` — 工具调用开始
- `OnEnd` — 工具调用完成
- `OnError` — 工具调用出错
- 统一通过 Eino 全局回调注册，替代 executor 内手动处理
