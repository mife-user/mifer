package executor

import (
	"context"
	"fmt"
	"mifer/internal/domain"
	"strings"
)

func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(content string) error) error {
	// 添加用户消息到记忆中
	e.Humen.Memory.AppendUser(req.Content)

	// 运行对话
	iter := e.Runner.Run(c, e.Humen.Memory.Messages)

	lastMsg := &strings.Builder{}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return event.Err
		}

		message := event.Output.MessageOutput.Message
		if message == nil {
			continue
		}

		lastMsg.WriteString(message.Content)

		if err := callback(message.Content); err != nil {
			return err
		}
	}

	if lastMsg.String() == "" {
		return fmt.Errorf("未收到助手回复")
	}

	// 添加助手消息到记忆中
	e.Humen.Memory.AppendAssistant(lastMsg.String())
	if err := e.Humen.Memory.Save(); err != nil {
		return err
	}
	return nil
}
