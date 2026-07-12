package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	aicallback "mifer/internal/ai/callback"
	"mifer/internal/ai/confirm"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ErrPlanRejected 用户拒绝计划时返回此错误。
var ErrPlanRejected = errors.New("plan rejected by user")

// PlanInstruction PlanAgent 的系统指令。
const PlanInstruction = `你是 Mifer 计划制定助手，具备只读分析能力。

可用工具（只读，无写入/执行权限）：
- file_reader / file_viewer — 读取和查看项目文件
- web_search / web_fetch — 搜索互联网资料
- knowledge_search — 检索知识库

职责：
1. 分析用户需求、项目上下文和对话历史
2. 制定可执行的详细计划，每一步具体到文件路径、操作类型、预期结果
3. 输出纯文本 Markdown 格式计划内容
4. 如有不确定的信息，标注"需确认"

限制：
- 你只能查看代码和搜索资料来分析问题
- 你无法执行任何写入或命令操作
- 计划中描述的可执行操作将在后续由主 Agent 执行

输出格式：
## 计划概述
<一句话描述>

## 步骤
### 1. <步骤名>
- 操作: <描述>
- 工具: <将使用的工具名>
- 涉及文件: <文件路径>`

// planGraphResult 计划图中间节点的输出结构。
type planGraphResult struct {
	FilePath string // 计划文件路径
	Content  string // 计划内容
}

// newPlanGraph 创建计划图：PlanAgent(只读) → 写入文件 → 用户确认。
func newPlanGraph(ctx context.Context, planAgent adk.Agent, store *confirm.Store) compose.Runnable[[]*schema.Message, string] {
	g := compose.NewGraph[[]*schema.Message, string]()

	// ── plan_agent: 运行 PlanAgent（流式），用 aicallback handler 处理工具事件 ──
	if err := g.AddLambdaNode("plan_agent", compose.InvokableLambda(
		func(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {
			agentInput := &adk.AgentInput{
				Messages:        msgs,
				EnableStreaming: true,
			}
			cb, hasCB := confirm.CallbackFromCtx(ctx)
			logger.Info("plan_agent 开始运行",
				logger.S("has_cb", fmt.Sprintf("%v", hasCB)),
				logger.S("streaming", "true"))

			// 用 aicallback handler 处理工具事件（与 Mifer 正常对话一致）
			var runOpts []adk.AgentRunOption
			if cb != nil {
				toolCB := aicallback.NewHandler(func(event, content string) error {
					return cb(event, content)
				})
				runOpts = append(runOpts, adk.WithCallbacks(toolCB))
			}

			iter := planAgent.Run(ctx, agentInput, runOpts...)
			var contents []string
			var currentAgent string
			var eventCount, agentFwd, responseFwd, thinkingFwd int
			var promptTokens, completionTokens, totalTokens int

			if cb != nil {
				_ = cb("agent_start", "PlanAgent")
			}

			for {
				event, ok := iter.Next()
				if !ok {
					break
				}
				eventCount++
				if event.Err != nil {
					return nil, fmt.Errorf("PlanAgent 执行出错: %w", event.Err)
				}

				// 转发 agent 切换事件
				if event.AgentName != "" && event.AgentName != currentAgent {
					if currentAgent != "" && cb != nil {
						_ = cb("agent_end", currentAgent)
					}
					currentAgent = event.AgentName
					if cb != nil {
						_ = cb("agent_start", currentAgent)
						agentFwd++
					}
				}

				// 工具事件已由 aicallback handler 处理，此处只处理消息输出
				if event.Output == nil || event.Output.MessageOutput == nil {
					continue
				}
				msgOutput := event.Output.MessageOutput

				// ── 流式消息：逐 chunk 转发 thinking / response / token ──
				if msgOutput.IsStreaming && msgOutput.MessageStream != nil {
					for {
						chunk, chunkErr := msgOutput.MessageStream.Recv()
						if chunkErr != nil {
							if !errors.Is(chunkErr, io.EOF) {
								logger.Warn("plan_agent 流式读取异常", logger.C(chunkErr))
							}
							break
						}
						if chunk.ReasoningContent != "" && cb != nil {
							_ = cb("thinking", chunk.ReasoningContent)
							thinkingFwd++
						}
						if chunk.Content != "" {
							contents = append(contents, chunk.Content)
							if cb != nil {
								_ = cb("response", chunk.Content)
								responseFwd++
							}
						}
						if chunk.ResponseMeta != nil && chunk.ResponseMeta.Usage != nil {
							u := chunk.ResponseMeta.Usage
							promptTokens += u.PromptTokens
							completionTokens += u.CompletionTokens
							totalTokens += u.TotalTokens
							if cb != nil {
								payload := fmt.Sprintf("%d\x00%d\x00%d\x00%d\x00%d",
									promptTokens, completionTokens, totalTokens,
									u.PromptTokenDetails.CachedTokens,
									u.CompletionTokensDetails.ReasoningTokens)
								_ = cb("token", payload)
							}
						}
					}
				}
			}

			if currentAgent != "" && cb != nil {
				_ = cb("agent_end", currentAgent)
			}
			if cb != nil {
				_ = cb("agent_end", "PlanAgent")
			}

			logger.Info("plan_agent 事件迭代完成",
				logger.I("total_events", eventCount),
				logger.I("content_parts", len(contents)),
				logger.I("forwarded_agent", agentFwd),
				logger.I("forwarded_response", responseFwd),
				logger.I("forwarded_thinking", thinkingFwd))

			if len(contents) == 0 {
				return nil, errors.New("PlanAgent 未生成有效计划")
			}
			return schema.AssistantMessage(strings.Join(contents, ""), nil), nil
		},
	)); err != nil {
		logger.Warn("创建 plan_agent 节点失败", logger.C(err))
		return nil
	}

	// ── plan_write: 写入 .mifer/plans/ ──
	if err := g.AddLambdaNode("plan_write", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (planGraphResult, error) {
			plansDir := filepath.Join(conf.GetConfig().Path.Workdir, ".mifer", "plans")
			if err := os.MkdirAll(plansDir, 0755); err != nil {
				return planGraphResult{}, fmt.Errorf("创建 plans 目录失败: %w", err)
			}

			name := fmt.Sprintf("plan_%s.md", time.Now().Format("20060102_150405"))
			planPath := filepath.Join(plansDir, name)

			if err := os.WriteFile(planPath, []byte(msg.Content), 0644); err != nil {
				return planGraphResult{}, fmt.Errorf("写入计划文件失败: %w", err)
			}

			logger.Debug("计划文件已写入", logger.S("path", planPath), logger.I("len", len(msg.Content)))
			return planGraphResult{FilePath: planPath, Content: msg.Content}, nil
		},
	)); err != nil {
		logger.Warn("创建 plan_write 节点失败", logger.C(err))
		return nil
	}

	// ── plan_confirm: SSE 确认 + 阻塞等用户 ──
	if err := g.AddLambdaNode("plan_confirm", compose.InvokableLambda(
		func(ctx context.Context, result planGraphResult) (string, error) {
			cb, ok := confirm.CallbackFromCtx(ctx)
			if !ok || cb == nil {
				logger.Warn("plan_confirm 节点未找到 callback，跳过用户确认")
				return result.Content, nil
			}

			sessionID, _ := confirm.SessionIDFromCtx(ctx)

			confirmID := fmt.Sprintf("plan_%d", time.Now().UnixNano())
			argsJSON, _ := json.Marshal(map[string]string{
				"file_path": result.FilePath,
				"content":   result.Content,
			})
			entry := &confirm.PendingEntry{
				ID:        confirmID,
				ToolName:  "plan",
				Arguments: string(argsJSON),
				ResultCh:  make(chan confirm.ConfirmResult, 1),
				CreatedAt: time.Now(),
				SessionID: sessionID,
			}
			store.Add(entry)

			eventJSON, _ := json.Marshal(map[string]string{
				"id":        confirmID,
				"file_path": result.FilePath,
				"content":   result.Content,
			})
			if err := cb("plan_confirm", string(eventJSON)); err != nil {
				logger.Warn("发送 plan_confirm 事件失败", logger.C(err))
				store.Remove(confirmID)
				return result.Content, nil
			}

			select {
			case res := <-entry.ResultCh:
				if res.Approved {
					logger.Debug("计划已确认", logger.S("plan", result.FilePath))
					return result.Content, nil
				}
				logger.Debug("计划被拒绝", logger.S("action", res.Action))
				return "", ErrPlanRejected
			case <-time.After(30 * time.Minute):
				store.Remove(confirmID)
				logger.Warn("计划确认超时")
				return "", ErrPlanRejected
			case <-ctx.Done():
				store.Remove(confirmID)
				return "", ctx.Err()
			}
		},
	)); err != nil {
		logger.Warn("创建 plan_confirm 节点失败", logger.C(err))
		return nil
	}

	_ = g.AddEdge(compose.START, "plan_agent")
	_ = g.AddEdge("plan_agent", "plan_write")
	_ = g.AddEdge("plan_write", "plan_confirm")
	_ = g.AddEdge("plan_confirm", compose.END)

	compiled, err := g.Compile(ctx)
	if err != nil {
		logger.Warn("编译计划图失败", logger.C(err))
		return nil
	}
	return compiled
}
