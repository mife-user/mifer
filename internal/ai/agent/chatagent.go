package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// newChatAgent 创建一个聊天agent
func newChatAgent(c context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	agent, err := adk.NewChatModelAgent(c, &adk.ChatModelAgentConfig{
		Name:          "MiTalker",
		Description:   "日常交流专家，负责与用户进行友好、自然的对话",
		Instruction:   " 你是MiTalker，用户的日常交流伙伴。\n\n能力范围：\n- 回答各类知识性问题\n- 提供建议和思路\n- 进行轻松愉快的闲聊\n\n交流风格：\n- 语气友好、自然、亲切\n- 回答简洁有条理\n- 不确定时坦诚说明，不编造事实\n- 遇到需要文件操作、命令执行、安全审查等任务时，引导用户向管理员说明需求",
		Model:         chatModel,
		MaxIterations: 600,
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}
