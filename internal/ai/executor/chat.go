package executor

import (
	"context"
	"mifer/internal/domain"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"strings"
)

func (e *Executor) Chat(c context.Context, req *domain.TalkReq, callback func(content string) error) error {
	e.Humen.Memory.AppendUser(req.Content)

	iter := e.Runner.Run(c, e.Humen.Memory.Messages)

	lastMsg := &strings.Builder{}
	eventCount := 0
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++
		if event.Err != nil {
			logger.Error("AI事件错误", logger.C(event.Err))
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

	logger.Debug("AI事件迭代完成", logger.I("eventCount", eventCount), logger.I("msgLen", lastMsg.Len()))

	if lastMsg.String() == "" {
		return errorer.New(errorer.ErrCallBackNull)
	}

	e.Humen.Memory.AppendAssistant(lastMsg.String())
	if err := e.Humen.Memory.Save(); err != nil {
		return err
	}
	return nil
}
