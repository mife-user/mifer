# Eino 框架函数方法参考文档

> 基于 `github.com/cloudwego/eino v0.8.13`，本项目实际使用的 API 子集。

---

## 目录

1. [adk — Agent 开发套件](#1-adk--agent-开发套件)
2. [deep — DeepAgent 编排器](#2-deep--deepagent-编排器)
3. [schema — 消息类型与辅助函数](#3-schema--消息类型与辅助函数)
4. [model — 聊天模型接口](#4-model--聊天模型接口)
5. [tool — 工具接口定义](#5-tool--工具接口定义)
6. [utils — 工具推断辅助](#6-utils--工具推断辅助)
7. [compose — 编排配置](#7-compose--编排配置)
8. [eino-ext 模型初始化](#8-eino-ext-模型初始化)
9. [RAG 检索增强生成](#9-rag-检索增强生成)
   - 9.1 [schema.Document — 文档数据类型](#91-schemadocument--文档数据类型)
   - 9.2 [embedding.Embedder — 文本向量化](#92-embeddingembedder--文本向量化)
   - 9.3 [indexer.Indexer — 文档存储入库](#93-indexerindexer--文档存储入库)
   - 9.4 [retriever.Retriever — 文档检索召回](#94-retrieverretriever--文档检索召回)
   - 9.5 [document.Loader — 文档加载读取](#95-documentloader--文档加载读取)
   - 9.6 [document.Transformer — 文档转换处理](#96-documenttransformer--文档转换处理)
   - 9.7 [prompt.ChatTemplate — 提示词模板](#97-promptchattemplate--提示词模板)
   - 9.8 [RAG 完整流程示例](#98-rag-完整流程示例)

---

## 1. adk — Agent 开发套件

### 1.1 `adk.NewChatModelAgent`

【函数签名】
```go
func NewChatModelAgent(ctx context.Context, config *ChatModelAgentConfig) (*ChatModelAgent, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文，超时/取消控制 |
| 入参 | config | `*ChatModelAgentConfig` | Agent 配置结构体 |
| 出参 | — | `*ChatModelAgent` | 创建的 Agent 实例，实现 `adk.Agent` 接口 |
| 出参 | — | `error` | 创建失败时返回错误 |

**ChatModelAgentConfig 结构体：**
```go
type ChatModelAgentConfig struct {
    Name         string              // Agent 名称，用于日志和事件标识
    Description  string              // 简要描述，供编排器调度时参考
    Instruction  string              // 系统提示词，定义 Agent 的行为和能力边界
    Model        model.BaseChatModel // 底层 LLM 模型实例
    ToolsConfig  ToolsConfig         // 工具配置（可选），内含 compose.ToolsNodeConfig
    MaxIterations int                // 最大迭代次数，超出后强制终止
    SubAgents    []Agent             // 子 Agent 列表（仅编排器 Agent 使用）
}
```

【示例】创建无工具的对话 Agent：
```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:         "MiTalker",
    Description:  "日常交流专家",
    Instruction:  "你是MiTalker，用户的日常交流伙伴…",
    Model:        chatModel,       // model.BaseChatModel 实例
    MaxIterations: 5,
})
```

创建带工具的 Agent：
```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "MiCommander",
    Description: "终端命令执行专家",
    Instruction: "你是MiCommander，在安全沙箱中执行shell命令…",
    Model:       chatModel,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: tools.CommandTools(cfg), // []tool.BaseTool
        },
    },
    MaxIterations: 5,
})
```

【注意事项】
- `Name` 必须唯一，同一编排器下不可重复。
- `MaxIterations` 建议设置 3~10，防止 Agent 陷入无限循环消耗 Token。
- `ToolsConfig.ToolsNodeConfig.Tools` 仅对当前 Agent 可见，不同 Agent 之间工具隔离。
- 返回的 `*ChatModelAgent` 实现了 `adk.Agent` 接口，可直接作为 `SubAgents` 元素。

---

### 1.2 `adk.NewRunner`

【函数签名】
```go
func NewRunner(ctx context.Context, config RunnerConfig) *Runner
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | config | `RunnerConfig` | Runner 配置 |
| 出参 | — | `*Runner` | Runner 实例，用于执行 Agent |

**RunnerConfig 结构体：**
```go
type RunnerConfig struct {
    Agent           Agent // 要执行的 Agent 实例（通常是编排器 Agent）
    EnableStreaming bool  // 是否启用流式输出
}
```

【示例】
```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           humen.Agent,  // deep.New 创建的编排器 Agent
    EnableStreaming: true,
})
```

【注意事项】
- `NewRunner` **不返回 error**，创建总是成功。
- `EnableStreaming: true` 时，消息输出通过 `MessageStream` 逐块读取；为 `false` 时直接获取完整 `Message`。
- Runner 是无状态的，可复用多次调用 `Run()`。

---

### 1.3 `Runner.Run`

【函数签名】
```go
func (r *Runner) Run(ctx context.Context, messages []*schema.Message) *stream.Reader[*events.Message]
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | messages | `[]*schema.Message` | 对话历史消息列表，包含用户和助手的历史消息 |
| 出参 | — | `*stream.Reader[*events.Message]` | 事件流迭代器，逐个产出 Agent 事件 |

【示例】完整的事件消费循环：
```go
iter := runner.Run(ctx, memory.Messages) // []*schema.Message

for {
    event, ok := iter.Next()
    if !ok {
        break // 事件流结束
    }
    if event.Err != nil {
        return event.Err
    }

    // 检测 Agent 切换
    if event.AgentName != "" {
        log.Printf("当前Agent: %s", event.AgentName)
    }

    if event.Output == nil || event.Output.MessageOutput == nil {
        continue
    }

    msgOutput := event.Output.MessageOutput

    // 非流式消息
    if !msgOutput.IsStreaming {
        if msgOutput.Role == schema.Assistant {
            fmt.Print(msgOutput.Message.Content)
        }
        if msgOutput.Role == schema.Tool {
            log.Printf("工具结果 [%s]: %s", msgOutput.ToolName, msgOutput.Message.Content)
        }
        continue
    }

    // 流式消息 — 逐块读取
    for {
        chunk, err := msgOutput.MessageStream.Recv()
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            return err
        }
        // 推理内容（如 Claude 的 extended thinking）
        if chunk.ReasoningContent != "" {
            log.Printf("[思考] %s", chunk.ReasoningContent)
        }
        // 普通文本
        fmt.Print(chunk.Content)
    }
}
```

【注意事项】
- `messages` 是整个对话历史，每次调用时需包含之前所有的用户/助手消息。
- 必须消费到 `iter.Next()` 返回 `false`，否则底层资源可能泄漏。
- 事件中 `Err` 字段非 nil 表示执行错误，应立即终止。
- 流式消息（`IsStreaming == true`）的 `MessageStream.Recv()` 返回 `io.EOF` 表示该流结束，需继续外层 `Next()` 获取后续事件。
- `AgentName` 可用于检测多 Agent 编排中的切换（agent_start / agent_end）。
- `ReasoningContent` 仅在支持 extended thinking 的模型（如 Claude Opus）中非空。

---

### 1.4 `adk.Agent` 接口

【函数签名】
```go
type Agent interface {
    // (内部接口，由 adk.ChatModelAgent 和 deep.Agent 实现)
}
```

【示例】作为类型使用：
```go
type Humen struct {
    Agent  adk.Agent
    Memory *memory.Memory
}

// deep.Config 中指定子 Agent
SubAgents: []adk.Agent{chatAgent, editerAgent, summarizerAgent}
```

【注意事项】
- 该接口由 `*adk.ChatModelAgent` 和 `deep.New` 返回的 Agent 自动实现。
- 用户无需手动实现此接口。
- 主要作为 `SubAgents` 切片的元素类型和 `RunnerConfig.Agent` 字段类型。

---

### 1.5 `adk.ToolsConfig`

```go
type ToolsConfig struct {
    ToolsNodeConfig    compose.ToolsNodeConfig
    EmitInternalEvents bool // 是否将子 Agent 内部事件转发到父级事件流
}
```

【注意事项】
- `EmitInternalEvents: true` 使得编排器可以监听到子 Agent 的工具调用事件，用于 UI 侧边栏显示。

---

## 2. deep — DeepAgent 编排器

### 2.1 `deep.New`

【函数签名】
```go
func New(ctx context.Context, config *Config) (Agent, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | config | `*Config` | 编排器配置 |
| 出参 | — | `Agent` | 编排器 Agent 实例（实现 adk.Agent） |
| 出参 | — | `error` | 创建失败时返回错误 |

**Config 结构体：**
```go
type Config struct {
    Name         string              // Agent 名称
    Description  string              // 简要描述
    Instruction  string              // 系统提示词，定义调度策略
    ChatModel    model.BaseChatModel // 调度器使用的 LLM
    ToolsConfig  adk.ToolsConfig     // 工具配置
    SubAgents    []adk.Agent         // 子 Agent 列表
    MaxIteration int                 // 最大迭代次数
}
```

【示例】
```go
agent, err := deep.New(ctx, &deep.Config{
    Name:        "Mifer",
    Description: "智能任务编排器，根据用户请求自动选择最合适的专家Agent处理任务",
    Instruction: `你是Mifer智能助手的管理员，负责分析用户请求并调度合适的专家Agent。

你可以调用的专家Agent：
- MiTalker：日常对话交流
- MiEditer：文件读取、写入、创建
- MiSummarizer：文档阅读与摘要总结
- MiPlanner：项目计划与方案编写
- MiCommander：安全执行终端命令
- MiAuditor：代码与配置安全审计

工作原则：
1. 先理解用户意图，再选择合适的Agent
2. 复杂任务可串联多个Agent协作完成
3. 涉及安全操作时优先咨询MiAuditor
4. 回复用户时使用中文，简洁清晰`,
    ChatModel:   registry.Get("default"),
    ToolsConfig: adk.ToolsConfig{
        EmitInternalEvents: true,
    },
    SubAgents:    []adk.Agent{chatAgent, editerAgent, summarizerAgent, plannerAgent, commanderAgent, auditorAgent},
    MaxIteration: 5,
})
```

【注意事项】
- `deep.New` 基于 Eino ADK 的预构建编排器，自动处理任务分解和 Agent 调度。
- `ChatModel` 是编排器的"大脑"，负责路由决策，建议使用较强的模型（如 Claude Sonnet/Opus）。
- `SubAgents` 中的 Agent 必须通过 `adk.NewChatModelAgent` 创建。
- `MaxIteration` 限制编排器与子 Agent 交互的最大轮次，防止死循环。
- `Instruction` 中的子 Agent 描述应与各 Agent 的 `Description` 保持一致，便于模型正确路由。

---

## 3. schema — 消息类型与辅助函数

### 3.1 `schema.Message` 结构体

```go
type Message struct {
    Role      RoleType   // schema.User / schema.Assistant / schema.System / schema.Tool
    Content   string     // 消息文本内容
    ToolCalls []ToolCall // 工具调用请求（Assistant 消息中）
}
```

【示例】作为对话历史的存储单元：
```go
type Memory struct {
    Messages []*schema.Message
}

// JSON 序列化/反序列化（与 JSONL 文件交互）
var msg schema.Message
json.Unmarshal(line, &msg)

// 判断消息角色
if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
    // 这是一条工具调用请求
}
```

【注意事项】
- `ToolCalls` 仅在 `Role == schema.Assistant` 且模型请求工具调用时非空。
- JSON tag 支持标准序列化，可直接存入 JSONL 文件。

---

### 3.2 `schema.RoleType` 常量

```go
const (
    User      RoleType = "user"
    Assistant RoleType = "assistant"
    System    RoleType = "system"
    Tool      RoleType = "tool"
)
```

【示例】
```go
switch msgOutput.Role {
case schema.Assistant:
    // 助手回复
case schema.Tool:
    // 工具执行结果
}
```

---

### 3.3 `schema.UserMessage`

【函数签名】
```go
func UserMessage(content string) *Message
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | content | `string` | 用户消息文本 |
| 出参 | — | `*Message` | Role=User 的 Message 实例 |

【示例】
```go
memory.Messages = append(memory.Messages, schema.UserMessage("你好，请帮我总结这份报告"))
```

【注意事项】
- 创建的 Message 的 `Role` 固定为 `schema.User`。
- `ToolCalls` 为 nil。
- 线程安全由调用方保证（本项目使用 `sync.Mutex` 保护）。

---

### 3.4 `schema.AssistantMessage`

【函数签名】
```go
func AssistantMessage(content string, toolCalls []ToolCall) *Message
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | content | `string` | 助手消息文本 |
| 入参 | toolCalls | `[]ToolCall` | 工具调用列表，纯文本回复时传 nil |
| 出参 | — | `*Message` | Role=Assistant 的 Message 实例 |

【示例】
```go
// 纯文本回复
memory.Messages = append(memory.Messages, schema.AssistantMessage("报告总结如下…", nil))

// 包含工具调用
memory.Messages = append(memory.Messages, schema.AssistantMessage("", toolCalls))
```

【注意事项】
- `toolCalls` 为 `nil` 时表示纯文本回复。
- 通常模型通过 `event.Output.MessageOutput.Message` 返回的消息已包含正确的 `ToolCalls`，直接追加即可。

---

## 4. model — 聊天模型接口

### 4.1 `model.BaseChatModel` 接口

【函数签名】
```go
type BaseChatModel interface {
    // (Eino 内部接口，供 ChatModelAgent 调用)
}
```

【示例】作为类型使用：
```go
type Registry struct {
    models map[string]model.BaseChatModel
}

func (r *Registry) Get(name string) model.BaseChatModel {
    if m, ok := r.models[name]; ok {
        return m
    }
    return r.models["default"] // fallback
}
```

【注意事项】
- 该接口由 `openai.NewChatModel`、`claude.NewChatModel`、`gemini.NewChatModel`、`ollama.NewChatModel` 等返回的实例实现。
- 不需要直接调用其方法，交给 `adk.ChatModelAgent` 内部使用即可。
- 项目中通过 `Registry` 按名称索引不同的模型实力，实现多后端管理。

---

## 5. tool — 工具接口定义

### 5.1 `tool.BaseTool` 接口

```go
type BaseTool interface {
    Info() *ToolInfo
}
```

【示例】作为工具切片的元素类型：
```go
func FileTools() []tool.BaseTool {
    var tools []tool.BaseTool
    fr, _ := filereader.New()
    tools = append(tools, fr)
    fw, _ := filewriter.New()
    tools = append(tools, fw)
    return tools
}
```

### 5.2 `tool.InvokableTool` 接口

```go
type InvokableTool interface {
    BaseTool
    Invoke(ctx context.Context, input string) (string, error)
}
```

【示例】作为工具构造函数的返回类型：
```go
func New() (tool.InvokableTool, error) {
    return utils.InferTool("file_reader", "安全读取本地文本文件", readFile)
}
```

【注意事项】
- `InvokableTool` 是 `BaseTool` 的子接口，增加了 `Invoke` 方法。
- `utils.InferTool` 返回的就是 `InvokableTool`。
- 传入 `compose.ToolsNodeConfig.Tools` 时类型为 `[]tool.BaseTool`，`InvokableTool` 自动向上转型。

---

## 6. utils — 工具推断辅助

### 6.1 `utils.InferTool`

【函数签名】
```go
func InferTool(name, description string, fn any) (tool.InvokableTool, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | name | `string` | 工具名称，模型通过此名称调用工具 |
| 入参 | description | `string` | 工具描述，模型据此判断何时使用该工具 |
| 入参 | fn | `any` | 工具实现函数，必须满足 `func(context.Context, Input) (Output, error)` 签名 |
| 出参 | — | `tool.InvokableTool` | 生成的工具实例 |
| 出参 | — | `error` | 创建失败时返回错误 |

【示例】从普通 Go 函数创建工具：
```go
type FileReaderInput struct {
    FilePath  string `json:"file_path" jsonschema:"required,description=要读取的文件路径"`
    StartLine int    `json:"start_line" jsonschema:"description=起始行号（1-based），默认1"`
    MaxLines  int    `json:"max_lines" jsonschema:"description=最大读取行数，默认100，上限500"`
}

type FileReaderOutput struct {
    Content   string `json:"content"`
    Error     string `json:"error,omitempty"`
}

func readFile(ctx context.Context, input FileReaderInput) (FileReaderOutput, error) {
    // 实现文件读取逻辑…
    return FileReaderOutput{Content: "...", Error: ""}, nil
}

func New() (tool.InvokableTool, error) {
    return utils.InferTool(
        "file_reader",
        "安全读取本地文本文件内容，支持指定起始行号、行数限制和路径安全校验。",
        readFile,
    )
}
```

【注意事项】
- `fn` 的函数签名**必须**严格为 `func(context.Context, InputStruct) (OutputStruct, error)`。
- Input/Output 结构体的字段 `jsonschema` tag 决定了 LLM 看到的参数 Schema，务必写清楚 `required` 和 `description`。
- `description` 要准确描述工具功能，直接影响模型是否在正确场景调用该工具。
- 如果函数内部需要外部依赖（如配置），通过**闭包**注入：
  ```go
  func New(cfg *conf.Config) (tool.InvokableTool, error) {
      execute := func(ctx context.Context, input MyInput) (MyOutput, error) {
          return doWork(ctx, input, cfg) // cfg 通过闭包传入
      }
      return utils.InferTool("my_tool", "...", execute)
  }
  ```

---

## 7. compose — 编排配置

### 7.1 `compose.ToolsNodeConfig`

```go
type ToolsNodeConfig struct {
    Tools []tool.BaseTool // 工具列表
}
```

【示例】
```go
compose.ToolsNodeConfig{
    Tools: tools.FileTools(), // []tool.BaseTool
}
```

【注意事项】
- 不直接使用，总是嵌入在 `adk.ToolsConfig` 中。
- `Tools` 字段接受 `[]tool.BaseTool`，`InvokableTool` 可自动向上转型。

---

## 8. eino-ext 模型初始化

### 8.1 `openai.NewChatModel`

【函数签名】
```go
func NewChatModel(ctx context.Context, config *ChatModelConfig) (model.BaseChatModel, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | config | `*ChatModelConfig` | OpenAI 兼容配置 |
| 出参 | — | `model.BaseChatModel` | 模型实例 |
| 出参 | — | `error` | 错误 |

```go
type ChatModelConfig struct {
    Model   string // 模型名称，如 "gpt-4o"、"deepseek-chat"
    BaseURL string // API 地址，如 "https://api.openai.com/v1"
    APIKey  string // API 密钥
}
```

【示例】
```go
model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    Model:   "gpt-4o",
    BaseURL: "https://api.openai.com/v1",
    APIKey:  os.Getenv("OPENAI_API_KEY"),
})
```

【注意事项】
- 兼容所有 OpenAI API 格式的服务（DeepSeek、通义千问、本地 vLLM 等）。
- `APIKey` 为空时返回错误。
- 可通过 `BaseURL` 指向兼容 OpenAI 协议的第三方服务。

---

### 8.2 `claude.NewChatModel`

【函数签名】
```go
func NewChatModel(ctx context.Context, config *Config) (model.BaseChatModel, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | config | `*Config` | Claude 配置 |
| 出参 | — | `model.BaseChatModel` | 模型实例 |
| 出参 | — | `error` | 错误 |

```go
type Config struct {
    Model     string // 模型名称，如 "claude-sonnet-4-6"
    APIKey    string // Anthropic API 密钥
    MaxTokens int    // 最大输出 Token
}
```

【示例】
```go
model, err := claude.NewChatModel(ctx, &claude.Config{
    Model:     "claude-sonnet-4-6",
    APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
    MaxTokens: 4096,
})
```

【注意事项】
- 支持 Anthropic 原生 API，包括 extended thinking（推理内容通过 `chunk.ReasoningContent` 获取）。
- `MaxTokens` 建议不超过模型上限（Claude 3.5/4 系列通常为 4096 或 8192）。

---

### 8.3 `gemini.NewChatModel`

【函数签名】
```go
func NewChatModel(ctx context.Context, config *Config) (model.BaseChatModel, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | config | `*Config` | Gemini 配置 |
| 出参 | — | `model.BaseChatModel` | 模型实例 |
| 出参 | — | `error` | 错误 |

```go
type Config struct {
    Client *genai.Client // Google GenAI 客户端
    Model  string        // 模型名称
}
```

【示例】
```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
if err != nil {
    return nil, err
}
model, err := gemini.NewChatModel(ctx, &gemini.Config{
    Client: client,
    Model:  "gemini-2.0-flash",
})
```

【注意事项】
- 需要先创建 `genai.Client`（Google 官方 SDK），再传入。
- `genai.Client` 的生命周期由调用方管理。
- 注意 Gemini API Key 与 OpenAI/Claude 的格式不同。

---

### 8.4 `ollama.NewChatModel`

【函数签名】
```go
func NewChatModel(ctx context.Context, config *ChatModelConfig) (model.BaseChatModel, error)
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | config | `*ChatModelConfig` | Ollama 配置 |
| 出参 | — | `model.BaseChatModel` | 模型实例 |
| 出参 | — | `error` | 错误 |

```go
type ChatModelConfig struct {
    Model   string // 模型名称，如 "llama3.1"
    BaseURL string // Ollama 服务地址，如 "http://localhost:11434"
}
```

【示例】
```go
model, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
    Model:   "llama3.1",
    BaseURL: "http://localhost:11434",
})
```

【注意事项】
- Ollama 本地服务，无需 API Key。
- `BaseURL` 为空时默认 `http://localhost:11434`。
- 需确保本地 Ollama 服务已启动且模型已 pull。

---

## 9. RAG 检索增强生成

Eino 为 RAG（Retrieval-Augmented Generation）提供了完整的数据流水线组件，从文档加载到检索召回一应俱全。

**数据流向：** `Loader → Transformer → Indexer ↔ Retriever → ChatTemplate → ChatModel`
```
                  ┌───────────┐
  外部数据源  →   │  Loader   │   读取原始内容
                  └─────┬─────┘
                        │  []*schema.Document
                  ┌─────▼─────┐
                  │Transformer│   切分 / 过滤 / 重排
                  └─────┬─────┘
                        │  []*schema.Document
                  ┌─────▼─────┐
                  │  Indexer  │   向量化 + 存储
                  └─────┬─────┘
                        │
              ┌─────────┴─────────┐
      读取    │    Vector Store   │
              └─────────┬─────────┘
                        │
                  ┌─────▼─────┐
  用户查询  →    │ Retriever │   向量检索 + 召回
                  └─────┬─────┘
                        │  []*schema.Document（上下文）
                  ┌─────▼──────┐
                  │ChatTemplate│   拼装 Prompt（query + context）
                  └─────┬──────┘
                        │  []*schema.Message
                  ┌─────▼─────┐
                  │ ChatModel │   生成回答
                  └───────────┘
```

### 9.1 `schema.Document` — 文档数据类型

【函数签名】
```go
type Document struct {
    ID       string         `json:"id"`        // 文档唯一标识
    Content  string         `json:"content"`   // 文档文本内容
    MetaData map[string]any `json:"meta_data"` // 元数据（分数、向量、来源等）
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 属性 | ID | `string` | 后端分配的唯一标识，Indexer.Store 返回 |
| 属性 | Content | `string` | 文档文本内容 |
| 属性 | MetaData | `map[string]any` | 开放式元数据 map，跨 pipeline 阶段传递信息 |

**内置 MetaData 访问器：**

| 方法 | 说明 |
|------|------|
| `doc.Score() float64` | 获取检索相关性分数 |
| `doc.WithScore(s float64) *Document` | 设置相关性分数 |
| `doc.DenseVector() []float64` | 获取稠密向量 |
| `doc.WithDenseVector(v []float64) *Document` | 设置稠密向量 |
| `doc.SparseVector() map[int]float64` | 获取稀疏向量 |
| `doc.WithSparseVector(v map[int]float64) *Document` | 设置稀疏向量 |
| `doc.SubIndexes() []string` | 获取子索引分区名 |
| `doc.WithSubIndexes(idx []string) *Document` | 设置子索引分区名 |
| `doc.DSLInfo() map[string]any` | 获取 DSL 信息 |
| `doc.WithDSLInfo(info map[string]any) *Document` | 设置 DSL 信息 |
| `doc.ExtraInfo() string` | 获取额外信息 |
| `doc.WithExtraInfo(info string) *Document` | 设置额外信息 |

【示例】
```go
doc := &schema.Document{
    ID:      "doc_001",
    Content: "Eino 是 CloudWeGo 开源的 AI 应用开发框架…",
}
doc.WithScore(0.95).WithSubIndexes([]string{"tech", "golang"})

// 从 Retriever 返回的结果中读取元数据
docs, _ := retriever.Retrieve(ctx, "什么是 Eino？")
for _, d := range docs {
    fmt.Printf("分数: %.2f, 内容: %s\n", d.Score(), d.Content)
    fmt.Printf("子索引: %v\n", d.SubIndexes())
}
```

【注意事项】
- `Document` 是 Loader、Transformer、Indexer、Retriever 之间的**共享货币**，所有组件都通过它交换数据。
- Transformer 实现应该**保留**已有 MetaData 并**合并**新键，而不是整体替换 map，避免丢失上游阶段的信息。
- 向量字段（DenseVector/SparseVector）由 Embedder 填充，跨 Indexer/Retriever 必须使用**相同的 Embedder 模型**。

---

### 9.2 `embedding.Embedder` — 文本向量化

【函数签名】
```go
type Embedder interface {
    EmbedStrings(ctx context.Context, texts []string, opts ...Option) ([][]float64, error)
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | texts | `[]string` | 待向量化的文本批次 |
| 入参 | opts | `...Option` | 可选参数，如 `WithModel("text-embedding-3-small")` |
| 出参 | — | `[][]float64` | embeddings[i] = texts[i] 的向量 |
| 出参 | — | `error` | 错误 |

**Option 函数：**
```go
func WithModel(model string) Option  // 指定嵌入模型名称
```

【示例】独立使用 Embedder：
```go
emb, err := openai.NewEmbedder(ctx, &openai.EmbedderConfig{
    Model:  "text-embedding-3-small",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

vectors, err := emb.EmbedStrings(ctx, []string{
    "Eino 是 AI 应用框架",
    "Go 语言的 Agent 开发",
})
// vectors[0] → []float64{0.12, -0.34, 0.56, ...}  (1536维)
// vectors[1] → []float64{0.08, -0.41, 0.33, ...}  (1536维)
```

将 Embedder 注入到 Indexer / Retriever：
```go
// Indexer 端：写入时自动向量化
idx.Store(ctx, docs, indexer.WithEmbedding(emb))

// Retriever 端：查询时自动向量化
ret.Retrieve(ctx, "什么是 Eino？", retriever.WithEmbedding(emb))
```

【注意事项】
- `embeddings[i]` 按顺序对应 `texts[i]`，**顺序不会错乱**。
- 向量维度由模型决定（如 OpenAI ada-002 为 1536 维，text-embedding-3-large 为 3072 维），同一批次所有向量维度相同。
- Indexer 和 Retriever 必须使用**完全相同的 Embedder 模型**，否则向量空间不匹配，相似度计算无意义。
- 具体实现位于 `github.com/cloudwego/eino-ext/components/embedding/`（OpenAI、Ollama、Ark 等）。

---

### 9.3 `indexer.Indexer` — 文档存储入库

【函数签名】
```go
type Indexer interface {
    Store(ctx context.Context, docs []*schema.Document, opts ...Option) (ids []string, err error)
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | docs | `[]*schema.Document` | 待存储的文档批次 |
| 入参 | opts | `...Option` | 可选参数 |
| 出参 | — | `[]string` | 后端为每个文档分配的唯一 ID |
| 出参 | — | `error` | 错误 |

**Option 函数：**
```go
func WithEmbedding(emb embedding.Embedder) Option  // 注入 Embedder，存储前自动生成向量
func WithSubIndexes(subIndexes []string) Option     // 写入指定的子分区
```

【示例】
```go
idx, err := milvus.NewIndexer(ctx, &milvus.IndexerConfig{
    Collection: "knowledge_base",
    Host:       "localhost:19530",
})

docs := []*schema.Document{
    {ID: "1", Content: "Eino 支持 OpenAI、Claude 等多种模型后端…"},
    {ID: "2", Content: "MCP 协议允许 Agent 调用外部工具…"},
}

// 入库时自动向量化
ids, err := idx.Store(ctx, docs,
    indexer.WithEmbedding(embedder),
    indexer.WithSubIndexes([]string{"tech", "ai"}),
)
// ids → ["1", "2"]（后端分配的唯一标识）
```

【注意事项】
- Indexer 是 RAG 流水线的**写路径**（write path），与 Retriever（读路径）配对使用。
- `WithEmbedding` 注入 Embedder 后，Store 会自动调用 `EmbedStrings` 生成向量并存入文档的 `DenseVector` 元数据中。
- `WithSubIndexes` 可将文档存入逻辑子分区，便于后续按分区检索。
- 具体实现：Milvus、VikingDB、Elasticsearch、Redis 等，位于 `github.com/cloudwego/eino-ext/components/indexer/`。

---

### 9.4 `retriever.Retriever` — 文档检索召回

【函数签名】
```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, opts ...Option) ([]*schema.Document, error)
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | query | `string` | 自然语言查询字符串 |
| 入参 | opts | `...Option` | 可选参数 |
| 出参 | — | `[]*schema.Document` | 按相关性降序排列的文档列表 |
| 出参 | — | `error` | 错误 |

**Option 函数：**
```go
func WithEmbedding(emb embedding.Embedder) Option       // 查询向量化使用的 Embedder
func WithTopK(topK int) Option                           // 返回结果数量上限
func WithScoreThreshold(threshold float64) Option         // 最低相关性分数阈值（低于此值的文档被过滤）
func WithIndex(index string) Option                       // 指定检索的索引名称
func WithSubIndex(subIndex string) Option                 // 指定检索的子分区
func WithDSLInfo(dsl map[string]any) Option              // 透传 DSL 查询信息
```

【示例】
```go
ret, err := redis.NewRetriever(ctx, &redis.RetrieverConfig{
    Addr:     "localhost:6379",
    Password: "",
})

// 独立检索（手动传入 Embedder）
docs, err := ret.Retrieve(ctx, "什么是 Eino？",
    retriever.WithEmbedding(embedder),
    retriever.WithTopK(5),
    retriever.WithScoreThreshold(0.7),
    retriever.WithSubIndex("tech"),
)
// docs 按 Score 降序排列

for _, doc := range docs {
    fmt.Printf("[%.3f] %s\n", doc.Score(), doc.Content)
}
```

嵌入 Graph 中使用：
```go
// 将 Retriever 作为图的一个节点
graph.AddRetrieverNode("knowledge_search", ret)
```

【注意事项】
- Retriever 是 RAG 流水线的**读路径**（read path）。
- `WithScoreThreshold` 是**过滤**而非排序：分数低于阈值的文档被直接排除。
- `WithTopK` 限制返回数量，结果已按相关性降序排列。
- 若注入了 `WithEmbedding`，查询字符串会被自动向量化后检索；否则用原始文本检索（如 Elasticsearch 的全文搜索）。
- Embedder 必须与 Indexer 入库时使用的**模型完全一致**。
- 具体实现：Redis、Milvus、Elasticsearch、VikingDB 等，位于 `github.com/cloudwego/eino-ext/components/retriever/`。
- 本项目已引入 `redis.NewRetriever`，代码位于 `pkg/res/init.go`（待激活）。

---

### 9.5 `document.Loader` — 文档加载读取

【函数签名】
```go
type Loader interface {
    Load(ctx context.Context, src Source, opts ...LoaderOption) ([]*schema.Document, error)
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | src | `Source` | 文档来源（URI 标识） |
| 入参 | opts | `...LoaderOption` | 可选参数（如解析器配置） |
| 出参 | — | `[]*schema.Document` | 加载的文档列表 |
| 出参 | — | `error` | 错误 |

```go
type Source struct {
    URI string // 本地文件路径或远程 URL
}
```

**LoaderOption 函数：**
```go
func WithParserOptions(opts ...parser.Option) LoaderOption  // 配置格式解析器（PDF、Markdown、TXT 等）
```

【示例】
```go
loader, err := file.NewLoader(ctx, &file.LoaderConfig{
    // 文件加载器配置…
})

docs, err := loader.Load(ctx, document.Source{
    URI: "./docs/eino-guide.md",
}, document.WithParserOptions(
    parser.WithChunkSize(500),       // 每个文档块 500 字符
    parser.WithChunkOverlap(50),     // 块之间重叠 50 字符
))
// docs → [{Content: "Eino 是 CloudWeGo…", MetaData: {"source": "./docs/eino-guide.md"}}, …]
```

【注意事项】
- Loader 负责从外部源**读取原始字节**，实际的格式解析（PDF/Markdown/TXT）通常由 `parser.Parser` 完成。
- `Source.URI` 可以是本地路径或远程 URL（HTTP/S3/OSS 等，取决于具体 Loader 实现）。
- 建议在 MetaData 中至少保留 `source` URI，以便下游节点追踪文档来源。
- 具体实现：File、URL、S3、OSS 等，位于 `github.com/cloudwego/eino-ext/components/document/`。

---

### 9.6 `document.Transformer` — 文档转换处理

【函数签名】
```go
type Transformer interface {
    Transform(ctx context.Context, src []*schema.Document, opts ...TransformerOption) ([]*schema.Document, error)
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | src | `[]*schema.Document` | 输入文档列表 |
| 入参 | opts | `...TransformerOption` | 可选参数 |
| 出参 | — | `[]*schema.Document` | 转换后的文档列表 |
| 出参 | — | `error` | 错误 |

【示例】
```go
// 文本切分
splitter, err := recursive.NewSplitter(ctx, &recursive.SplitterConfig{
    ChunkSize:    500,
    ChunkOverlap: 50,
})

chunks, err := splitter.Transform(ctx, docs)
// 一篇长文档 → 多个重叠的文本块

// 去重过滤
deduper, err := dedup.NewTransformer(ctx, &dedup.Config{})
uniqueDocs, err := deduper.Transform(ctx, chunks)
```

【注意事项】
- Transformer 执行**切分、过滤、合并、重排**等操作，常位于 Loader 之后、Indexer 之前。
- 实现必须**保留已有 MetaData** 并**合并**新键，避免丢失来源、分数等上游信息。
- 常见的 Transformer：文本切分器（Recursive/Semantic）、去重器、重排序器等。
- 具体实现位于 `github.com/cloudwego/eino-ext/components/document/`。

---

### 9.7 `prompt.ChatTemplate` — 提示词模板

【函数签名】
```go
type ChatTemplate interface {
    Format(ctx context.Context, vs map[string]any, opts ...Option) ([]*schema.Message, error)
}
```

| 方向 | 名称 | 类型 | 说明 |
|------|------|------|------|
| 入参 | ctx | `context.Context` | 上下文 |
| 入参 | vs | `map[string]any` | 模板变量键值对 |
| 入参 | opts | `...Option` | 可选参数 |
| 出参 | — | `[]*schema.Message` | 渲染后的消息列表，可直接传入 ChatModel |
| 出参 | — | `error` | 变量缺失等模板错误 |

**构造函数：**
```go
func FromMessages(formatType schema.FormatType, templates ...schema.MessagesTemplate) *DefaultChatTemplate
```

**支持的模板语法（FormatType）：**
| 常量 | 说明 |
|------|------|
| `schema.FString` | `{variable}` 简单替换 |
| `schema.GoTemplate` | Go `text/template` 语法，支持条件和循环 |
| `schema.Jinja2` | Jinja2 模板语法 |

**消息占位符：**
```go
func MessagesPlaceholder(key string, optional bool) MessagesTemplate
```

【示例】基础模板：
```go
tmpl := prompt.FromMessages(schema.FString,
    schema.SystemMessage("你是一个乐于助人的助手，根据以下参考资料回答问题。"),
    schema.UserMessage("参考资料：{context}\n\n问题：{query}"),
)

// 渲染模板
msgs, err := tmpl.Format(ctx, map[string]any{
    "context": "Eino 是 CloudWeGo 开源的 AI 应用开发框架…",
    "query":   "什么是 Eino？",
})
// msgs → []*schema.Message{SystemMsg, UserMsg（变量已替换）}
```

带对话历史的模板：
```go
tmpl := prompt.FromMessages(schema.FString,
    schema.SystemMessage("你是一个乐于助人的助手，根据以下参考资料回答问题。\n参考资料：{context}"),
    schema.MessagesPlaceholder("history", true),  // 动态插入历史消息列表
    schema.UserMessage("{query}"),
)

msgs, err := tmpl.Format(ctx, map[string]any{
    "context": "Eino 是 CloudWeGo 开源…",
    "history": historyMsgs,      // []*schema.Message — 之前的对话历史
    "query":   "什么是 Eino？",
})
```

【注意事项】
- 模板中引用了但 `vs` 中**不存在的变量**会返回**运行时错误**，没有编译时检查，建议统一变量命名规范。
- `MessagesPlaceholder` 允许在模板的固定位置动态插入消息列表（如对话历史），第二个参数 `optional=true` 表示允许不传该 key。
- 模板通常位于 ChatModel 之前，在 Graph/Chain 中通过 `compose.WithOutputKey` 将前序节点输出转换为 `map[string]any`。
- 三种模板语法递进增强：`FString` 最简、`GoTemplate` 支持逻辑控制流、`Jinja2` 支持完整模板语法。

---

### 9.8 RAG 完整流程示例

以下是一个完整的 RAG 流水线，从文档加载到生成回答：

```go
// 1. 创建 Embedder（Indexer 和 Retriever 共用）
emb, _ := openai.NewEmbedder(ctx, &openai.EmbedderConfig{
    Model:  "text-embedding-3-small",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

// 2. 创建 Indexer（写路径）
idx, _ := redis.NewIndexer(ctx, &redis.IndexerConfig{
    Addr: "localhost:6379",
})

// 3. 创建 Retriever（读路径，与 Indexer 共享同一个 Embedder）
ret, _ := redis.NewRetriever(ctx, &redis.RetrieverConfig{
    Addr: "localhost:6379",
})

// 4. 加载文档
loader, _ := file.NewLoader(ctx, &file.LoaderConfig{})
docs, _ := loader.Load(ctx, document.Source{URI: "./knowledge/*.md"})

// 5. 切分文档
splitter, _ := recursive.NewSplitter(ctx, &recursive.SplitterConfig{
    ChunkSize: 500, ChunkOverlap: 50,
})
chunks, _ := splitter.Transform(ctx, docs)

// 6. 入库存储（自动向量化）
ids, _ := idx.Store(ctx, chunks, indexer.WithEmbedding(emb))

// 7. 创建提示词模板（注入检索结果）
tmpl := prompt.FromMessages(schema.FString,
    schema.SystemMessage("根据以下参考资料回答问题：\n{context}"),
    schema.UserMessage("{query}"),
)

// 8. 构建查询 → 检索 → 生成 的 Graph
graph := compose.NewGraph[map[string]any, *schema.Message]()
_ = graph.AddRetrieverNode("retrieve", ret, retriever.WithEmbedding(emb))
_ = graph.AddChatTemplateNode("prompt", tmpl)
_ = graph.AddChatModelNode("generate", chatModel)
_ = graph.AddEdge("retrieve", "prompt")
_ = graph.AddEdge("prompt", "generate")

// 9. 编译并运行
run, _ := graph.Compile(ctx)
result, _ := run.Invoke(ctx, map[string]any{
    "query": "Eino 支持哪些模型后端？",
})
fmt.Println(result.Content)
```

---

## 附录：事件流数据结构

`Runner.Run()` 返回的 `stream.Reader[*events.Message]` 中每个事件的核心字段：

| 字段路径 | 类型 | 说明 |
|----------|------|------|
| `.Err` | `error` | 事件错误，非 nil 时应立即终止 |
| `.AgentName` | `string` | 当前执行的 Agent 名称，变化时表示 Agent 切换 |
| `.Output` | `*events.Output` | 输出容器 |
| `.Output.MessageOutput` | `*events.MessageOutput` | 消息输出 |
| `.MessageOutput.IsStreaming` | `bool` | 是否为流式输出 |
| `.MessageOutput.Role` | `schema.RoleType` | 消息角色 |
| `.MessageOutput.ToolName` | `string` | 工具名称（仅 Tool 角色） |
| `.MessageOutput.Message` | `*schema.Message` | 完整消息（非流式） |
| `.MessageOutput.MessageStream` | `*stream.Reader[*schema.Chunk]` | 流式消息流 |
| `.Message.ToolCalls` | `[]schema.ToolCall` | 工具调用请求 |
| `.Message.Content` | `string` | 消息文本内容 |
| `.Chunk.ReasoningContent` | `string` | 推理/思考内容（extended thinking） |
| `.Chunk.Content` | `string` | 流式文本块 |
