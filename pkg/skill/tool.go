package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mifer/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const skillToolName = "skill"


// NewSkillTool 创建技能工具适配器
func NewSkillTool(manager *Manager, agentHub *AgentHub) *SkillTool {
	return &SkillTool{
		manager:  manager,
		agentHub: agentHub,
	}
}

// Info 返回技能工具的 ToolInfo，description 中包含所有可用技能列表
func (s *SkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info := &schema.ToolInfo{
		Name: skillToolName,
		Desc: s.buildDescription(),
	}

	params := map[string]*schema.ParameterInfo{
		"skill": {
			Type:     schema.String,
			Desc:     "要调用的技能名称",
			Required: true,
		},
	}
	info.ParamsOneOf = schema.NewParamsOneOfByParams(params)
	return info, nil
}

// buildDescription 构建包含技能列表的 tool description
func (s *SkillTool) buildDescription() string {
	var sb strings.Builder
	sb.WriteString("调用预定义的技能（Skill）。每个技能包含专业领域的操作指南和指令。")

	skills := s.manager.List()
	if len(skills) == 0 {
		sb.WriteString("\n当前没有可用技能。")
		return sb.String()
	}

	sb.WriteString("\n\n可用技能列表：")
	for _, sk := range skills {
		fmt.Fprintf(&sb, "\n- %s：%s", sk.Name, sk.Description)
		if sk.Context == "fork" && sk.Agent != "" {
			fmt.Fprintf(&sb, "（由 %s 独立执行）", sk.Agent)
		}
	}
	return sb.String()
}

// InvokableRun 执行技能
func (s *SkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Skill string `json:"skill"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析技能参数失败: %w", err)
	}

	skill, err := s.manager.Get(args.Skill)
	if err != nil {
		return fmt.Sprintf("技能 [%s] 不存在。可用技能: %v", args.Skill, s.skillNames()), nil
	}

	// 判断执行模式
	if skill.Context == "fork" && skill.Agent != "" && s.agentHub != nil {
		return s.runForkMode(ctx, skill)
	}

	// inline 模式：直接返回技能内容
	return s.runInlineMode(skill), nil
}

// runInlineMode 返回技能内容供 LLM 在当前上下文使用
func (s *SkillTool) runInlineMode(skill *Skill) string {
	return fmt.Sprintf(
		"【技能: %s】\n技能目录: %s\n\n%s",
		skill.Name,
		skill.BaseDir,
		skill.Content,
	)
}

// runForkMode 创建子 Agent 独立执行技能
func (s *SkillTool) runForkMode(ctx context.Context, skill *Skill) (string, error) {
	agent, err := s.agentHub.Get(skill.Agent)
	if err != nil {
		// Agent 不存在，降级为 inline
		logger.Warn(fmt.Sprintf("技能 [%s] fork 模式 Agent [%s] 不存在，降级为 inline", skill.Name, skill.Agent))
		return s.runInlineMode(skill), nil
	}

	// 构建子 Agent 输入
	userMsg := fmt.Sprintf("请严格遵循以下技能指令执行任务：\n\n技能名称: %s\n技能描述: %s\n\n%s",
		skill.Name, skill.Description, skill.Content,
	)

	input := &adk.AgentInput{
		Messages:        []adk.Message{schema.UserMessage(userMsg)},
		EnableStreaming: false,
	}

	// 执行子 Agent
	iter := agent.Run(ctx, input)

	var results []string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			logger.Error(fmt.Sprintf("技能 [%s] fork 执行错误", skill.Name), logger.C(event.Err))
			continue
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil && msg != nil && msg.Content != "" {
				results = append(results, msg.Content)
			}
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("技能 [%s] 执行完成，但未产生输出。", skill.Name), nil
	}

	return fmt.Sprintf("【技能 [%s] 由 %s 执行结果】\n\n%s", skill.Name, skill.Agent, strings.Join(results, "\n")), nil
}

// skillNames 返回所有技能名称列表
func (s *SkillTool) skillNames() []string {
	skills := s.manager.List()
	names := make([]string, len(skills))
	for i, sk := range skills {
		names[i] = sk.Name
	}
	return names
}
