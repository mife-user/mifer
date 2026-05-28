package executor

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"mifer/internal/ai/tools"
	"mifer/internal/ai/tools/filecreator"
	"mifer/pkg/logger"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var (
	errPlanRefused        = errors.New("计划被用户拒绝")
	errTooManySupplements = errors.New("补充次数超过限制")
)

// planRunner 计划 Graph 的运行器
//
// Graph 结构：analyze → plan → confirm → END
// 全部使用 Lambda 节点（string → string），内部调用 model，保持类型一致。
// confirm 节点内部处理 supplement 循环（最多 3 次）。
type planRunner struct {
	graph      compose.Runnable[string, string]
	model      model.BaseChatModel
	confirmBus *tools.ConfirmBus
	workdir    string
	sendSSE    func(event, data string) error
}

// newPlanRunner 创建计划 Graph 并编译
func newPlanRunner(ctx context.Context, chatModel model.BaseChatModel, confirmBus *tools.ConfirmBus, workdir string) (*planRunner, error) {
	pr := &planRunner{
		model:      chatModel,
		confirmBus: confirmBus,
		workdir:    workdir,
	}

	g := compose.NewGraph[string, string]()

	// 全部使用 Lambda 节点，类型统一为 string → string
	if err := g.AddLambdaNode("analyze", compose.InvokableLambda(pr.analyzeNode)); err != nil {
		return nil, fmt.Errorf("添加 analyze 节点失败: %w", err)
	}
	if err := g.AddLambdaNode("plan", compose.InvokableLambda(pr.planNode)); err != nil {
		return nil, fmt.Errorf("添加 plan 节点失败: %w", err)
	}
	if err := g.AddLambdaNode("confirm", compose.InvokableLambda(pr.confirmNode)); err != nil {
		return nil, fmt.Errorf("添加 confirm 节点失败: %w", err)
	}

	if err := g.AddEdge(compose.START, "analyze"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("analyze", "plan"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("plan", "confirm"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("confirm", compose.END); err != nil {
		return nil, err
	}

	compiled, err := g.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("编译计划 Graph 失败: %w", err)
	}
	pr.graph = compiled
	return pr, nil
}

// setSSECallback 设置 SSE 回调（每次 Chat 调用前设置）
func (pr *planRunner) setSSECallback(fn func(event, data string) error) {
	pr.sendSSE = fn
}

// analyzeNode 分析节点：接收任务描述，输出结构化的需求分析
func (pr *planRunner) analyzeNode(ctx context.Context, task string) (string, error) {
	msgs := []*schema.Message{
		schema.SystemMessage("你是一个需求分析专家。分析用户的任务，输出结构化的分析结果，包括：\n1. 任务目标\n2. 涉及的技术和组件\n3. 关键约束和注意事项\n4. 建议的实施路径"),
		schema.UserMessage(task),
	}
	resp, err := pr.model.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("需求分析失败: %w", err)
	}
	return resp.Content, nil
}

// planNode plan 节点：接收分析结果，生成计划并写入 .mifer/plans/
func (pr *planRunner) planNode(ctx context.Context, analysis string) (string, error) {
	msgs := []*schema.Message{
		schema.SystemMessage("你是一个项目计划专家。根据分析结果制定结构化的执行计划，输出 Markdown 格式。"),
		schema.UserMessage(fmt.Sprintf(
			"根据以下分析结果，制定一个详细的执行计划：\n\n%s\n\n"+
				"计划格式要求：\n"+
				"## 计划概述\n简要描述整体方案\n\n"+
				"## 执行步骤\n1. [步骤名称] — 描述、涉及文件、预估工作量\n2. ...\n\n"+
				"## 注意事项\n潜在风险及应对",
			analysis,
		)),
	}

	resp, err := pr.model.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("生成计划失败: %w", err)
	}
	planContent := resp.Content

	// 写入计划文件
	plansDir := filepath.Join(pr.workdir, ".mifer", "plans")
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		logger.Warn("创建计划目录失败", logger.C(err))
		return planContent, nil
	}

	taskHash := fmt.Sprintf("%x", md5.Sum([]byte(analysis)))[:8]
	filename := fmt.Sprintf("%s_%s.md", time.Now().Format("20060102_150405"), taskHash)
	filePath := filepath.Join(plansDir, filename)

	fc, err := filecreator.New()
	if err != nil {
		logger.Warn("创建 file_creator 失败", logger.C(err))
		return planContent, nil
	}
	fcArgs, _ := json.Marshal(map[string]string{
		"file_path": filePath,
		"content":   planContent,
	})
	if _, err := fc.InvokableRun(ctx, string(fcArgs)); err != nil {
		logger.Warn("写入计划文件失败", logger.S("path", filePath), logger.C(err))
	} else {
		logger.Info("计划文件已写入", logger.S("path", filePath))
	}

	if pr.sendSSE != nil {
		_ = pr.sendSSE("response", "计划已生成，路径: "+filePath+"\n\n")
	}

	return planContent, nil
}

// confirmNode confirm 节点：向 TUI 发送计划确认，内部处理 supplement 循环
func (pr *planRunner) confirmNode(ctx context.Context, planContent string) (string, error) {
	for i := 0; i < 3; i++ {
		result, err := pr.confirmBus.ConfirmPlan(ctx, planContent)
		if err != nil {
			return "", err
		}

		switch result.Action {
		case tools.ActionAccept:
			if pr.sendSSE != nil {
				_ = pr.sendSSE("response", "计划已确认，开始执行...\n\n")
			}
			return planContent, nil

		case tools.ActionRefuse:
			return "", errPlanRefused

		case tools.ActionSupplement:
			if pr.sendSSE != nil {
				_ = pr.sendSSE("response", fmt.Sprintf("收到补充意见: %s\n重新生成计划...\n\n", result.Input))
			}
			planContent, err = pr.replan(ctx, planContent, result.Input)
			if err != nil {
				return "", fmt.Errorf("重新生成计划失败: %w", err)
			}
		}
	}
	return "", errTooManySupplements
}

// replan 根据补充意见重新生成计划
func (pr *planRunner) replan(ctx context.Context, originalPlan, supplement string) (string, error) {
	msgs := []*schema.Message{
		schema.SystemMessage("你是一个项目计划专家。根据用户的补充意见修改计划，输出完整的 Markdown 格式计划。"),
		schema.UserMessage(fmt.Sprintf(
			"原计划：\n%s\n\n用户补充意见：%s\n\n请重新制定完整的计划。",
			originalPlan, supplement,
		)),
	}

	resp, err := pr.model.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
