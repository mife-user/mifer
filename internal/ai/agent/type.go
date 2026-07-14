package agent

import (
	"mifer/internal/ai/confirm"
	"mifer/internal/ai/llm"
	"mifer/internal/ai/prompt"
	"mifer/internal/ai/rag"
	"mifer/pkg/mcp"
	"mifer/pkg/skill"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// confirmMiddleware 包级工具确认中间件，由 Init() 设置后供 Agent 配置复用。
var confirmMiddleware compose.ToolMiddleware

// Agents 聚合三层 Agent：主编排器、QQ 通道、计划制定。
type Agents struct {
	Mifer adk.Agent // 主编排器（Orchestrator），nil 表示 api_key 未配置
	QQ    adk.Agent // QQ 通道专用（无工具，纯文本对话），nil 表示未启用
	Plan  adk.Agent // 计划制定（只读工具），nil 表示创建失败
}

// Graphs 聚合编译后的 Eino 可运行图。
type Graphs struct {
	Habit compose.Runnable[[]*schema.Message, string] // 用户习惯总结图
	Plan  compose.Runnable[[]*schema.Message, string] // 计划制定图
}

// Humen 用户 Agent 聚合结构体，包含所有 Agent、Graph 及运行时依赖。
type Humen struct {
	Agents       Agents
	Graphs       Graphs
	Prompt       *prompt.Prompty
	Registry     *llm.Registry
	MCPManager   *mcp.Manager
	SkillManager *skill.Manager
	ConfirmStore *confirm.Store // 工具确认存储
	AgentInfos   []AgentInfo    // Agent 元数据列表

	// 内部构造依赖（仅 Init 过程使用）
	ragSvc   rag.RAGService          // RAG 服务
	skillHub *skill.AgentHub         // Agent 注册中心
	errorMw  compose.ToolMiddleware  // 错误处理中间件
	persistenceMw compose.ToolMiddleware // 工具持久化中间件
}

// qqInstruction QQ 通道 Agent 的系统指令。
const qqInstruction = `你是 Mifer QQ 消息助手。

行为准则：
- 你是 QQ 聊天中的 AI 助手，直接回复用户问题
- 回复简洁自然，与用户消息长度匹配
- 使用与用户相同的语言
- 任务完成后直接结束，不追问
- 你没有文件、命令、搜索等工具，遇到需要这些能力的问题直接说明限制`

// HabitInstruction 用户习惯分析助手的系统指令，供 graph 调用方构建输入消息时使用。
const HabitInstruction = `你是用户习惯分析助手，根据对话内容分析并更新用户画像。

规则：
- 直接以 Markdown 格式输出完整的用户画像内容，不要添加额外说明
- 通过分析每轮对话来推断用户的偏好、习惯、技能和背景
- 如果已有现有画像，基于新对话增量更新，保留之前仍有价值的信息
- 包含但不限于：编程语言偏好、技术栈、工作习惯、常用工具、项目类型、沟通风格
- 内容简洁结构化，使用标题和列表组织
- 不记录敏感信息（密码、密钥、个人身份信息等）`

// miferInstruction Mifer 主 Agent 的系统指令，涵盖所有已注入工具的使用准则。
const miferInstruction = `你是 Mifer 智能助手，运行于 Windows 环境。你直接拥有以下全部工具——无需委派给其他 Agent，所有操作由你独立完成。

可用工具：
- file_reader / file_writer / file_creator / file_viewer / image_generator — 文件读写、创建、查看、图片生成
- command_executor — Windows PowerShell 5.1 命令执行
- knowledge_search / knowledge_store — 知识库检索与文档入库
- web_search / web_fetch — 互联网搜索与网页抓取
- qq_send_message — QQ 消息发送
- skill — 预定义技能调用

Windows 环境：
- 文件路径格式：C:\Users\xxx\file.txt 或 C:/Users/xxx/file.txt，相对路径亦可
- 终端为 PowerShell 5.1：不支持 && / ||，用 ; 分隔命令；环境变量用 $env:NAME
- 常用命令对照：find → Get-ChildItem -Recurse -Filter；grep → Select-String；wc -l → Measure-Object -Line；rm -rf → Remove-Item -Recurse -Force -Confirm:$false；mkdir -p → New-Item -ItemType Directory -Force；touch → New-Item -ItemType File

文件操作铁律（必须遵守）：
1. 写前必读 — 调用 file_writer / file_creator 之前必须先用 file_reader 确认文件当前状态
2. 创建前探测 — 用 file_reader 探测目标路径（不存在会返回错误），确认不存在后再用 file_creator
3. 基于实际修改 — 修改文件必须基于 file_reader 返回的实际内容，禁止凭空猜测

命令执行准则：
- command_executor 仅用于运行程序、构建、安装、测试等命令行操作
- 禁止用 command_executor 读取文件内容（cat/type/Get-Content）——读取文件始终用 file_reader
- 危险命令自动拒绝：递归删除项目外文件、系统关机、权限变更、下载并执行等

知识库准则：
- knowledge_store 存储文档资料（技术文档、会议纪要、需求说明），代码文件不要入库
- 遇到不熟悉的知识点优先用 knowledge_search 检索知识库

工作原则：
1. 先理解用户意图再行动，复杂任务分步执行
2. 涉及实时信息、新闻、最新资料时用 web_search；需要阅读具体网页内容时用 web_fetch
3. 涉及安全操作时审慎评估风险，必要时拒绝并说明原因
4. 工具操作失败时分析错误原因，最多尝试 2 次替代方案；连续 3 次失败后停止并向用户报告
5. 回复使用中文，简洁清晰`
