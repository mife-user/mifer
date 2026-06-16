package askuser

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mifer/internal/ai/question"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"
)

// AskUserInput 向用户提问的输入参数
type AskUserInput struct {
	Question string   `json:"question" jsonschema:"required,description=向用户提出的问题"`
	Options  []string `json:"options" jsonschema:"required,description=供用户选择的答案选项列表，至少2项"`
}

// AskUserOutput 用户回答的输出结果
type AskUserOutput struct {
	Question     string `json:"question"`
	Answer       string `json:"answer"`
	IsSupplement bool   `json:"is_supplement"`
	Error        string `json:"error,omitempty"`
}

// New 创建向用户提问工具
func New() (tool.InvokableTool, error) {
	return utils.InferTool("ask_user",
		"当需要明确用户需求或偏好时使用。向用户提出一个问题并提供选项供选择，"+
			"等待用户回答后返回结果。适用于：方案选择、技术选型、功能优先级排序等场景。"+
			"用户始终可以选择「补充说明」来输入自定义回答。",
		askUser)
}

// askUser 阻塞等待用户回答
func askUser(ctx context.Context, input AskUserInput) (AskUserOutput, error) {
	store := question.GetStore(ctx)
	if store == nil {
		return AskUserOutput{Error: "问题存储未初始化"}, nil
	}

	callback := question.GetCallback(ctx)
	if callback == nil {
		return AskUserOutput{Error: "事件回调未初始化"}, nil
	}

	sessionID := question.GetSessionID(ctx)

	// 校验输入
	if input.Question == "" {
		return AskUserOutput{Error: "问题不能为空"}, nil
	}
	if len(input.Options) < 2 {
		return AskUserOutput{Error: "至少需要2个选项"}, nil
	}

	// 构建待回答问题项
	entry := &question.QuestionEntry{
		ID:        uuid.New().String(),
		Question:  input.Question,
		Options:   input.Options,
		ResultCh:  make(chan question.QuestionResult, 1),
		CreatedAt: time.Now(),
		SessionID: sessionID,
	}

	store.Add(entry)
	defer store.Remove(entry.ID)

	// 发送 SSE 事件通知 TUI 展示问题
	event := question.AskUserEvent{
		ID:       entry.ID,
		Question: input.Question,
		Options:  input.Options,
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		logger.Error("序列化问题事件失败", logger.C(err))
		return AskUserOutput{Error: fmt.Sprintf("序列化失败: %v", err)}, nil
	}

	if err := callback("ask_user", string(eventJSON)); err != nil {
		logger.Error("发送问题事件失败", logger.C(err))
		return AskUserOutput{Error: fmt.Sprintf("发送事件失败: %v", err)}, nil
	}

	// 阻塞等待用户回答或超时
	timeoutSec := 300 // 默认 5 分钟超时
	select {
	case result := <-entry.ResultCh:
		if result.Answer == "" && !result.IsSupplement {
			return AskUserOutput{
				Question: input.Question,
				Error:    "用户取消或超时",
			}, nil
		}
		logger.Debug("用户已回答问题",
			logger.S("id", entry.ID),
			logger.S("answer", result.Answer))
		return AskUserOutput{
			Question:     input.Question,
			Answer:       result.Answer,
			IsSupplement: result.IsSupplement,
		}, nil

	case <-time.After(time.Duration(timeoutSec) * time.Second):
		logger.Warn("用户回答问题超时", logger.S("id", entry.ID))
		return AskUserOutput{
			Question: input.Question,
			Error:    "等待用户回答超时",
		}, nil

	case <-ctx.Done():
		logger.Debug("问题等待被取消", logger.S("id", entry.ID))
		return AskUserOutput{
			Question: input.Question,
			Error:    "上下文已取消",
		}, ctx.Err()
	}
}
