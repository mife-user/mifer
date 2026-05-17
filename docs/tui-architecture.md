# Mifer TUI 架构文档

## 目录

1. [框架概述](#1-框架概述)
2. [文件结构](#2-文件结构)
3. [启动流程](#3-启动流程)
4. [Bubble Tea 事件循环](#4-bubble-tea-事件循环)
5. [消息流详解](#5-消息流详解)
6. [渲染管线](#6-渲染管线)
7. [组件体系](#7-组件体系)
8. [交互设计](#8-交互设计)
9. [数据流图](#9-数据流图)

***

## 1. 框架概述

TUI 基于 **Bubble Tea**（Charmbracelet 出品）构建，遵循 **Elm Architecture** 模式：

```
┌──────────┐    Msg     ┌──────────┐    Cmd     ┌──────────┐
│  Runtime  │ ────────→ │ Update() │ ────────→ │  Runtime  │
│ (事件源)  │           │ (状态机)  │           │ (执行器)  │
└──────────┘           └──────────┘           └──────────┘
                             │                      │
                             │ 新状态                │ 新 Msg
                             ↓                      ↓
                        ┌──────────┐           ┌──────────┐
                        │  View()  │           │ Update() │
                        │ (渲染)    │           │          │
                        └──────────┘           └──────────┘
                             │
                             │ 终端字符串
                             ↓
                        终端输出
```

**核心概念：**

- **Model** — 持有全部 UI 状态的结构体
- **Init()** — 返回初始命令（如光标闪烁）
- **Update(msg)** — 接收消息，更新状态，返回新命令
- **View()** — 根据当前状态生成终端 UI 字符串
- **tea.Cmd** — 异步命令（`func() tea.Msg`），由框架在后台执行

**关键特性：**

- **非阻塞**：所有耗时操作（HTTP 请求、定时器）封装为 `tea.Cmd`，框架在后台 goroutine 执行
- **单向数据流**：消息从事件源流入 Update，状态变化后驱动 View 重新渲染
- **即时响应**：UI 始终可交互，即使正在等待 AI 响应

***

## 2. 文件结构

```
cli/
├── cli.go                  # CLI 入口（RunTUI / Run REPL）
├── client/                 # HTTP API 客户端
│   ├── client.go           # Client 聚合（Chat / Memory / Excmem）
│   ├── chathandler/        # SSE 流式聊天
│   ├── memhandler/         # 对话记忆
│   └── excmemhandler/      # 记忆会话切换
├── render/                 # 渲染辅助
│   ├── lip/                # lipgloss 样式集合
│   │   ├── type.go         # Style 结构体（8 种子样式）
│   │   └── init.go         # 样式初始化（颜色降级）
│   └── mark/               # glamour markdown 渲染
│       ├── type.go         # Mark 结构体（双渲染器）
│       ├── init.go         # 初始化（dark + notty 降级）
│       └── rend.go         # Render() 渲染入口
└── tui/                    # TUI 核心（Bubble Tea）
    ├── type.go             # 消息类型 + Model 结构体
    ├── init.go             # NewModel() + Init()
    ├── update.go           # Update() + 按键处理 + 异步命令
    └── view.go             # View() 渲染管线
```

***

## 3. 启动流程

```
main() / serve 命令
  │
  └─→ bootstrap.NewApplication()
       └─→ initCli()
            └─→ cli.New(config)
                 └─→ client.New(baseURL)      // 创建 HTTP 客户端
                 └─→ tui.NewModel(client, config)  // 创建 TUI Model
                      │
                      ├─→ textarea.New()            // 输入组件
                      ├─→ spinner.New(MiniDot)      // 旋转动画（braille 点阵）
                      ├─→ viewport.New(0, 0)        // 滚动视口（初始 0×0）
                      │    └─→ Style = Background + Padding + RoundedBorder
                      ├─→ mark.Init()               // glamour 双渲染器
                      └─→ lip.Init(config)          // lipgloss 8 种子样式

cli.RunTUI()
  └─→ tea.NewProgram(model,
       tea.WithAltScreen(),           // 使用备用屏幕（退出后恢复原终端）
       tea.WithMouseCellMotion())     // 启用鼠标支持
  └─→ p.Run()                         // 进入事件循环
```

***

## 4. Bubble Tea 事件循环

```
p.Run()
  │
  ├─→ cmd := model.Init()                    // ① 初始命令（textarea.Blink）
  │
  └─→ for {                                  // ② 事件循环
        msg := waitForEvent()                //    等待事件（按键/定时器/HTTP响应）
        model, cmd := model.Update(msg)      //    ③ 更新状态
        print(model.View())                  //    ④ 渲染 UI
        if cmd == Quit { break }             //    ⑤ 检查退出
        go executeInBackground(cmd)          //    ⑥ 执行命令（生成下一轮 msg）
      }
```

**消息来源：**

| 事件类型      | 产生时机          | 消息类型                       |
| --------- | ------------- | -------------------------- |
| 终端 resize | 窗口尺寸变化        | `tea.WindowSizeMsg`        |
| 键盘输入      | 用户按键          | `tea.KeyMsg`               |
| 鼠标事件      | 滚轮/点击         | `tea.MouseMsg`             |
| 定时器       | `tea.Tick` 到期 | 用户自定义（如 `spinner.TickMsg`） |
| HTTP 响应   | SSE 流累积完成     | `chatRespMsg`              |
| 系统命令      | 记忆加载/切换       | `systemMsg`                |

***

## 5. 消息流详解

### 5.1 聊天消息完整流程

这是最核心的用户交互流程，涉及按键处理、HTTP 请求、动画和渲染。

```
用户输入 "你好" + Enter
  │
  ▼
tea.KeyMsg("enter")
  │
  ▼
Model.Update()
  └─→ handleEnter()
       │
       ├─→ ① 读取 textarea 内容: "你好"
       ├─→ ② 清空 textarea，重置高度为 1 行
       ├─→ ③ 记录到输入历史（去重）
       ├─→ ④ 追加 user message 到 messages 列表
       ├─→ ⑤ 设置 m.thinking = true
       ├─→ ⑥ 设置 m.needsAutoScroll = true
       │
       └─→ ⑦ 返回 tea.Batch(
              sendChatCmd(client, "你好"),   // HTTP SSE 请求
              spinner.Tick(),               // 启动旋转动画
            )
       │
       ├──────────────────────────────────────┐
       │                                      │
       ▼                                      ▼
  sendChatCmd 在 goroutine 中执行        spinner.Tick 定时器 (~83ms)
       │                                      │
       │  client.Chat.Send()                  │  TickMsg → Model.Update()
       │  ├─→ HTTP POST /api/ai/chat          │  ├─→ m.thinking == true
       │  ├─→ SSE 流式接收                    │  ├─→ spinner.Update(TickMsg)
       │  │   ├─ "thinking" event → 跳过      │  │   └─→ 帧+1，返回下一 Tick 命令
       │  │   └─ 其他 event → buf.WriteString  │  └─→ View() 渲染新帧
       │  └─→ SSE 流结束                       │
       │                                      │
       └─→ 返回 chatRespMsg{content, err}     │
              │                               │
              ▼                               │
       Model.Update()                         │
       ├─→ m.thinking = false  ←──────────────┘ (下一个 TickMsg 停止动画)
       ├─→ mark.Render(content)
       │    ├─→ glamour dark 主题渲染 markdown → ANSI
       │    └─→ 失败时降级到 notty（纯文本）
       ├─→ 追加 assistant message (含预渲染 ANSI)
       └─→ m.needsAutoScroll = true
              │
              ▼
       View()
       ├─→ 构建所有消息行（user + assistant）
       ├─→ thinking=false，不渲染 spinner
       ├─→ viewport.SetContent(所有行)
       ├─→ needsAutoScroll=true → viewport.GotoBottom()
       └─→ lipgloss.JoinVertical(viewport, textarea)
```

### 5.2 系统命令流程

```
用户输入 "/viewmemory" + Enter
  │
  ▼
handleEnter()
  └─→ 匹配 /viewmemory 前缀
       └─→ return loadMemoryCmd(client, id)
              │
              ▼
         HTTP GET /api/memory/:id
         └─→ 返回 systemMsg{content: "记忆内容"}
              │
              ▼
         Model.Update(systemMsg)
         ├─→ 追加 system message（青色）
         └─→ needsAutoScroll = true
```

### 5.3 历史导航流程

```
用户输入 "你好" 并发送
用户输入 "世界" 并发送
用户输入 "测试"
  │
  ▼ 按 ↑ (在首行)
handleHistoryUp()
  ├─→ historyIdx: -1 → len(history)-1 (指向 "世界")
  ├─→ pendingInput = "测试" (暂存)
  ├─→ textarea 显示 "世界"
  │
  ▼ 继续按 ↑
  ├─→ historyIdx: 1 → 0 (指向 "你好")
  ├─→ textarea 显示 "你好"
  │
  ▼ 按 ↓
  ├─→ historyIdx: 0 → 1 (指向 "世界")
  ├─→ textarea 显示 "世界"
  │
  ▼ 再按 ↓
  ├─→ historyIdx 已是最后 → historyIdx = -1
  ├─→ 恢复 pendingInput: textarea 显示 "测试"
  └─→ pendingInput = ""
```

### 5.4 Tab 补全流程

```
假设 CompletableCommands = ["/viewmemory", "/excmem", "exit", "help"]

用户输入 "/" + Tab
  │
  ▼
handleTabComplete()
  ├─→ findMatches("/") → ["/viewmemory", "/excmem"]
  ├─→ longestCommonPrefix → "/"
  ├─→ common == trimmed → 不修改 textarea
  ├─→ completions = ["/viewmemory", "/excmem"]
  ├─→ completionIdx = -1, completionBase = "/"
  │
  ▼ 再次 Tab
cycleCompletion()
  ├─→ completionIdx: -1 → 0
  ├─→ textarea = "/viewmemory "
  │
  ▼ 再次 Tab
cycleCompletion()
  ├─→ completionIdx: 0 → 1
  └─→ textarea = "/excmem "
```

***

## 6. 渲染管线

View() 每次 Update 后调用，生成终端 UI 字符串。渲染分 6 步：

### 第 1 步：门控检查

```
width == 0       → "正在启动..."（首个 WindowSizeMsg 到达前）
contentHeight < 1 → "窗口太小..."（终端高度不足）
```

### 第 2 步：构建消息行列表

```
messages = [
  {role: "user",      content: "你好"},
  {role: "assistant", content: "你好！...", rendered: "带ANSI的彩色文本"},
  {role: "system",    content: "记忆内容...\n多行"},
]

遍历后生成 msgLines:
  [0] "You: 你好"                         ← User 样式（绿色粗体）
  [1] "──────────────────"                ← 分隔线
  [2] "带ANSI的彩色文本第1行"              ← 从 glamour 预渲染拆行
  [3] "带ANSI的彩色文本第2行"
  [4] "──────────────────"                ← 分隔线
  [5] "记忆内容..."                       ← Sys 样式（青色）
  [6] "多行"
  [7] "──────────────────"                ← 分隔线
```

### 第 3 步：追加 thinking 行

```
if m.thinking:
  msgLines += ["⠋ Thinking..."]  ← Think 样式（橙色斜体）
  // ⠋ 字符每 ~83ms 切换到下一 braille 帧
```

### 第 4 步：追加错误行

```
if m.err != "":
  msgLines += ["错误: xxx"]  ← Err 样式（红色）
```

### 第 5 步：viewport 内容与自动滚底

```
content = strings.Join(msgLines, "\n")
viewport.SetContent(content)
// viewport 内部将 content 按 \n 拆分为行数组
// 计算 YOffset 范围，提供 AtTop()/AtBottom() 查询

if needsAutoScroll:
  viewport.GotoBottom()  // 将 YOffset 设为 maxYOffset
  needsAutoScroll = false
```

### 第 6 步：组合输出

```
┌─────────────────────────────────────┐
│ viewport.View()                     │  ← 背景色 + 圆角边框 + 内边距
│                                     │     只渲染可见行（基于 YOffset）
│  You: 你好                          │
│  ──────────────────                 │
│  AI 渲染后的回复内容...              │
│  ──────────────────                 │
│                                     │
├─────────────────────────────────────┤
│ textarea.View()                     │  ← 输入区域
│ 输入消息...                         │     占位符 + 光标 + 多行支持
└─────────────────────────────────────┘

lipgloss.JoinVertical(Top, viewport, textarea)
```

***

## 7. 组件体系

### 7.1 使用的 Bubbles 组件

| 组件         | 包                  | 职责                | Model 字段     |
| ---------- | ------------------ | ----------------- | ------------ |
| `textarea` | `bubbles/textarea` | 多行文本输入，占位符，光标管理   | `m.textarea` |
| `viewport` | `bubbles/viewport` | 消息区滚动容器，鼠标滚轮，内容裁剪 | `m.viewport` |
| `spinner`  | `bubbles/spinner`  | Braille 点阵旋转动画    | `m.spinner`  |

### 7.2 使用的自定义渲染组件

| 组件         | 包                        | 职责                     |
| ---------- | ------------------------ | ---------------------- |
| `glamour`  | `charmbracelet/glamour`  | Markdown → ANSI 终端彩色输出 |
| `lipgloss` | `charmbracelet/lipgloss` | 终端样式（前景色、边框、对齐、拼接）     |

### 7.3 自定义样式体系（`cli/render/lip/`）

```
base (Bold)
├── User     → 绿色 #00D787，粗体      ← "You: 消息"
├── AI       → 绿色 #00ff11            ← (当前未使用，glamour 自行着色)
├── Think    → 橙色 #FFB86C，斜体      ← "⠋ Thinking..."
├── Err      → 红色 #FF5555            ← "错误: xxx"
├── Sys      → 青色 #8BE9FD            ← 系统消息
├── Scroll   → 灰色 #666666            ← 滚动指示器（当前 viewport 替代）
├── Separator→ 灰色 #444444            ← 分隔线
└── Help     → 灰色 #888888            ← 帮助文本
```

### 7.4 颜色降级策略

所有子样式的颜色都通过 `getFg(configColor, fallback)` 获取：

- 优先使用配置文件中的颜色
- 配置为空时使用硬编码的降级颜色
- 确保在没有配置文件的情况下终端仍能正常显示

***

## 8. 交互设计

### 8.1 按键映射

| 按键       | 上下文          | 行为                |
| -------- | ------------ | ----------------- |
| `Enter`  | 任意           | 提交输入              |
| `Ctrl+N` | 任意           | 在 textarea 中插入换行符 |
| `↑`      | textarea 首行  | 历史导航（上一条）         |
| `↑`      | textarea 非首行 | textarea 光标上移     |
| `↓`      | textarea 末行  | 历史导航（下一条）         |
| `↓`      | textarea 非末行 | textarea 光标下移     |
| `Tab`    | 任意           | 命令补全              |
| `Ctrl+C` | 任意           | 退出程序              |
| `Esc`    | 任意           | 退出程序              |
| `鼠标滚轮`   | 消息区          | 上下滚动（每次 1 行）      |

### 8.2 命令列表

| 命令                 | 行为             |
| ------------------ | -------------- |
| `exit` / `quit`    | 退出程序           |
| `help`             | 显示帮助信息         |
| `/viewmemory [id]` | 查看指定/默认会话的对话记忆 |
| `/excmem <id>`     | 切换到指定记忆会话      |
| 其他文本               | 发送给 AI 的聊天消息   |

### 8.3 边界处理

- **Terminal 太小**：`contentHeight < 1` 时显示提示而非崩溃
- **AI 响应为空**：显示 "AI 返回了空内容" 错误
- **Markdown 渲染失败**：降级到原始文本显示
- **网络错误**：显示错误信息，不丢失历史消息
- **历史去重**：连续相同输入不重复记录
- **历史环形淘汰**：超过 MaxHistory 条数后丢弃最早记录
- **补全重置**：用户手动修改输入后自动退出补全模式

***

## 9. 数据流图

```
                          ┌──────────────┐
                          │  HTTP Server │
                          │  :8080       │
                          └──────┬───────┘
                                 │ SSE / JSON
                          ┌──────┴───────┐
                          │  client.Client│
                          │  Chat/Memory  │
                          └──────┬───────┘
                                 │ tea.Cmd (闭包)
                                 │
┌─────────────┐  tea.KeyMsg  ┌──┴──────────┐  tea.WindowSizeMsg  ┌────────────┐
│   Keyboard   │─────────────→│              │←───────────────────│  Terminal  │
└─────────────┘              │  Model       │                    └────────────┘
┌─────────────┐  tea.MouseMsg│  ┌─────────┐ │  ANSI String       ┌────────────┐
│   Mouse      │─────────────→│  │messages │ │───────────────────→│  Display   │
└─────────────┘              │  │thinking  │ │                    └────────────┘
                             │  │spinner   │ │
                             │  │viewport  │ │
                             │  │textarea  │ │
                             │  │history   │ │
                             │  │...       │ │
                             │  └─────────┘ │
                             └──────────────┘
                                    │
                               tea.Cmd (spinner.Tick)
                                    │
                              ┌─────┴─────┐
                              │  Spinner   │
                              │  ~83ms 定时│
                              └───────────┘

完整消息循环示例（用户说 "你好"）：

1. Keyboard → tea.KeyMsg("enter") → Model.Update()
2. handleEnter() → tea.Batch(sendChatCmd, spinner.Tick)
3. spinner.Tick → goroutine: 83ms 后 TickMsg → Model.Update()
4. Model.Update(TickMsg) → spinner 帧+1 → 返回下一个 spinner.Tick
5. View() 渲染新帧 "⠋ Thinking..."
6. HTTP SSE → 积累响应 → chatRespMsg → Model.Update()
7. thinking=false → 下一个 TickMsg 停止动画
8. mark.Render() → viewport.SetContent() → View() 渲染完整对话
```

