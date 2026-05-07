package executor

import (
	"context"
	"fmt"
	"mifer/internal/domain"

	"github.com/cloudwego/eino/schema"
)

func (e *Executor) Chat(c context.Context, req *domain.TalkReq) (*domain.TalkResp, error) {
	// 添加用户消息到记忆中
	e.Humen.Memory.AppendUser(req.Content)

	// 运行对话
	iter := e.Runner.Run(c, e.Humen.Memory.Messages)

	var lastMsg string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return nil, event.Err
		}
		// 只收集非流式的、角色为 Assistant 的消息
		if event.Output != nil && event.Output.MessageOutput != nil &&
			!event.Output.MessageOutput.IsStreaming &&
			event.Output.MessageOutput.Role == schema.Assistant {
			lastMsg = event.Output.MessageOutput.Message.Content
		}
	}

	if lastMsg == "" {
		return nil, fmt.Errorf("未收到助手回复")
	}

	// 添加助手消息到记忆中
	e.Humen.Memory.AppendAssistant(lastMsg)
	if err := e.Humen.Memory.Save(); err != nil {
		return nil, err
	}
	return &domain.TalkResp{Content: lastMsg}, nil
}
