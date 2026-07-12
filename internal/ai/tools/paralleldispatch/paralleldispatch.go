package paralleldispatch

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"mifer/pkg/logger"
	"mifer/pkg/skill"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const maxParallelTasks = 10

// ParallelTask 单个并行任务定义
type ParallelTask struct {
	Agent  string `json:"agent" jsonschema:"required,description=执行任务的Agent名称"`
	Prompt string `json:"prompt" jsonschema:"required,description=任务描述或提示词"`
}

// ParallelDispatchInput 并行调度工具入参
type ParallelDispatchInput struct {
	Tasks []ParallelTask `json:"tasks" jsonschema:"required,description=要并行执行的任务列表"`
}

// TaskResult 单个任务的执行结果
type TaskResult struct {
	Agent   string `json:"agent"`
	Success bool   `json:"success"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ParallelDispatchOutput 并行调度工具出参
type ParallelDispatchOutput struct {
	Results []TaskResult `json:"results"`
	Error   string       `json:"error,omitempty"`
}

// New 创建并行调度工具。
// agentHub 用于查找目标 Agent 实例，构造时从 hub 中提取可用 Agent 列表写入工具描述。
func New(agentHub *skill.AgentHub) (tool.InvokableTool, error) {
	desc := buildDescription(agentHub)

	dispatch := func(ctx context.Context, input ParallelDispatchInput) (ParallelDispatchOutput, error) {
		if len(input.Tasks) == 0 {
			return ParallelDispatchOutput{Error: "任务列表不能为空"}, nil
		}
		if len(input.Tasks) > maxParallelTasks {
			return ParallelDispatchOutput{Error: fmt.Sprintf("单次最多并行执行 %d 个任务", maxParallelTasks)}, nil
		}

		results := make([]TaskResult, len(input.Tasks))
		var wg sync.WaitGroup

		for i, task := range input.Tasks {
			wg.Add(1)
			go func(idx int, t ParallelTask) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						results[idx] = TaskResult{
							Agent:   t.Agent,
							Success: false,
							Error:   fmt.Sprintf("任务执行 panic: %v", r),
						}
					}
				}()

				agent, err := agentHub.Get(t.Agent)
				if err != nil {
					results[idx] = TaskResult{
						Agent:   t.Agent,
						Success: false,
						Error:   err.Error(),
					}
					return
				}

				agentInput := &adk.AgentInput{
					Messages:        []adk.Message{schema.UserMessage(t.Prompt)},
					EnableStreaming: false,
				}

				iter := agent.Run(ctx, agentInput)
				var contents []string
				for {
					event, ok := iter.Next()
					if !ok {
						break
					}
					if event.Err != nil {
						results[idx] = TaskResult{
							Agent:   t.Agent,
							Success: false,
							Error:   fmt.Sprintf("Agent 执行出错: %s", event.Err.Error()),
						}
						return
					}
					if event.Output != nil && event.Output.MessageOutput != nil {
						msgOutput := event.Output.MessageOutput
						message := msgOutput.Message
						if message == nil {
							continue
						}
						// 只收集最终的 AI 回复（纯文本，无工具调用），过滤中间消息
						if msgOutput.Role == schema.Assistant && len(message.ToolCalls) == 0 && message.Content != "" {
							contents = append(contents, message.Content)
						}
					}
				}

				results[idx] = TaskResult{
					Agent:   t.Agent,
					Success: true,
					Content: strings.Join(contents, "\n"),
				}
			}(i, task)
		}

		wg.Wait()
		return ParallelDispatchOutput{Results: results}, nil
	}

	t, err := utils.InferTool("parallel_dispatch", desc, dispatch)
	if err != nil {
		logger.Error("创建 parallel_dispatch 工具失败", logger.C(err))
		return nil, err
	}
	return t, nil
}

// buildDescription 构建包含可用 Agent 列表的工具描述
func buildDescription(agentHub *skill.AgentHub) string {
	var sb strings.Builder
	sb.WriteString("并行调度工具，同时向多个 Agent 分派独立任务并收集结果。")

	names := agentHub.Names()
	if len(names) == 0 {
		sb.WriteString("\n当前没有可用的 Agent，请先在配置文件中定义自定义 Agent。")
		return sb.String()
	}

	sb.WriteString("\n\n可用 Agent：")
	for _, name := range names {
		fmt.Fprintf(&sb, "\n- %s", name)
	}
	sb.WriteString("\n\n使用场景：需要多个 Agent 并行处理独立任务时（如同时读取多个文件、分别检索不同主题），一次性提交所有任务以提高效率。")
	return sb.String()
}
