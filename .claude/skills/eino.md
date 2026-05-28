---
name: eino
description: CloudWeGo Eino 框架开发指导 — Agent/Graph/Tool/Runner/多Agent 协作模式与最佳实践
---

# Eino 框架开发指导

基于 [Eino 框架详解文章](https://mife-user.github.io/posts/eino框架详解/) 和 Mifer 项目实践提炼。

## 核心概念速查

| 概念 | 作用 | 类比 |
|------|------|------|
| **ChatModel** | 封装大模型 API 调用 | 拨通 AI 的电话 |
| **Tool** | 让模型能执行操作（读文件、搜索等） | 给 AI 的工具箱 |
| **Agent** | 模型 + 工具 + ReAct 循环的打包 | 雇一个能自己干活的助手 |
| **Runner** | 管理对话生命周期、历史、事件流 | 助手的工作台 |
| **Graph** | 固定步骤的流水线（编译后执行） | 写死的操作手册 |
| **DeepAgent** | 主 Agent 自动调度子 Agent | 经理带团队 |

## 构建模式对比

```
开放式对话 → Agent + Runner        runner.Query() → 事件流（流式）
固定流程   → Graph                 compiled.Invoke() → 最终结果（一次性）
复杂任务   → DeepAgent/Graph+Agent  组合使用
```

### Agent 模式（模型自主决策）

```go
// 创建模型
model, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    Model: "gpt-4o-mini", APIKey: "...",
})

// 创建 Agent：模型 + 工具 + ReAct 循环
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "coder",
    Instruction: "你是编程助手",
    Model:       model,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{readFileTool, searchTool},
        },
    },
    MaxIterations: 20,
})

// 用 Runner 运行
runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
iter := runner.Query(ctx, "帮我分析 main.go")
for {
    event, ok := iter.Next()
    if !ok { break }
    fmt.Print(event.Message.Content)
}
```

### Graph 模式（你定义固定流程）

```go
graph := compose.NewGraph[string, string]()

// 添加节点（Lambda 节点或 ChatModel 节点）
graph.AddLambdaNode("read_file", func(ctx context.Context, path string) (string, error) {
    data, _ := os.ReadFile(path)
    return string(data), nil
})
graph.AddChatModelNode("ai_review", model)

// 固定路线
graph.AddEdge(compose.START, "read_file")
graph.AddEdge("read_file", "ai_review")
graph.AddEdge("ai_review", compose.END)

// 编译 + 执行
compiled, _ := graph.Compile(ctx)
result, _ := compiled.Invoke(ctx, "main.go")
```

### Graph 加分支

```go
graph.AddBranch("review", compose.NewGraphBranch(
    func(ctx context.Context, msg *schema.Message) (string, error) {
        if strings.Contains(msg.Content, "需要修改") {
            return "auto_fix", nil
        }
        return "format", nil
    },
    map[string]bool{"auto_fix": true, "format": true},
))
```

## 多 Agent 三种模式

### DeepAgent（推荐，开箱即用）

主 Agent 通过内置 `task` 工具自动委派子 Agent，上下文隔离。

```go
searchAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name: "SearchAgent", Description: "搜索代码和文档的专家",
    // ...
})

mainAgent, _ := deep.New(ctx, &deep.Config{
    Name:      "MainAgent",
    ChatModel: model,
    SubAgents: []adk.Agent{searchAgent, codeAgent},
    Instruction: `收到需求→分析→委派专家→汇总结果`,
})
```

### AgentTool（手动控制粒度）

```go
codeTool, _ := adk.AgentAsTool(ctx, codeAgent, "call_code_expert", "让编码专家写代码")
// 像普通 Tool 一样放进 Tools 列表
```

### Supervisor（监工模式，适合多步骤多轮调度）

子 Agent 完成后自动 Transfer 回报 Supervisor，Supervisor 决定下一步。

## Tool 实现

```go
type MyTool struct{}

func (t *MyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "my_tool",
        Desc: "工具描述，模型根据此描述决定何时调用",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "param1": {Type: schema.String, Desc: "参数说明", Required: true},
        }),
    }, nil
}

func (t *MyTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
    var p struct{ Param1 string }
    json.Unmarshal([]byte(args), &p)
    // 执行逻辑...
    return result, nil
}
```

## Graph 变成 Tool（GraphTool）

```go
compiled, _ := graph.Compile(ctx)
graphTool := adk.NewGraphTool(compiled, "graph_tool_name", "描述", adk.GraphToolCallOption{})
// 放进 Agent 的 Tools 列表，和普通 Tool 混用
```

## ReAct 循环原理

```
1. 把用户问题发给模型
2. 看模型返回：
   → 文字（无 ToolCalls） → 输出，结束
   → ToolCall → 执行工具 → 结果发回模型 → 回到第 2 步
```

Eino 的 `ChatModelAgent` 自动完成这个循环，`MaxIterations` 控制最大轮数。

## 流式输出

```go
stream, _ := model.Stream(ctx, messages)
for {
    chunk, err := stream.Recv()
    if errors.Is(err, io.EOF) { break }
    fmt.Print(chunk.Content) // 逐 token 输出
}
```

## Callback 注入

通过 Callback 监控 Agent 执行的每个环节（模型调用、工具执行、节点切换）。

## 关键注意事项

1. **Agent vs Graph 不是先后关系，是不同场景的选择** — 简单对话用 Agent，固定流程用 Graph，复杂任务用 DeepAgent
2. **Runner 只管一个 Agent**，但那个 Agent 内部可以是一个团队（DeepAgent）
3. **Graph 编译后执行** `graph.Compile()` → `compiled.Invoke()`，无对话历史，每次调用独立
4. **DeepAgent 子 Agent 上下文隔离** — 子 Agent 只收到委派任务描述，不看到完整对话历史
5. **Graph 可以变成 Tool**，Agent 也可以变成 Tool — 全部可以组合

## Mifer 项目中的实际使用

- `internal/ai/llm/` — ChatModel 多后端管理（Registry 模式）
- `internal/ai/agent/` — DeepAgent 编排（Mifer 主控 + MiEditer 子 Agent）
- `internal/ai/executor/` — Runner 包装，事件循环处理
- `internal/ai/tools/` — 自定义工具集
