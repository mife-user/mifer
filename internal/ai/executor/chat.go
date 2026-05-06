package executor

import (
	"context"
	"mifer/internal/domain"
)

func (e *Executor) Chat(c context.Context, req *domain.TalkReq) (*domain.TalkResp, error) {
	// 添加用户消息到记忆中
	e.Humen.Memory.Append(req.Content)
	// 运行对话
	sync := e.Runner.Run(c, e.Humen.Memory.Messages)

	event, ok := sync.Next()
	if !ok {
		return nil, nil
	}
	resp := event.Output.MessageOutput.Message.Content
	// 添加助手消息到记忆中
	e.Humen.Memory.Append(resp)
	err := e.Humen.Memory.Save()
	if err != nil {
		return nil, err
	}
	return &domain.TalkResp{Content: resp}, nil
}
