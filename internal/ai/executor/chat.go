package executor

import (
	"context"
	"errors"
	"io"
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

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msgOutput := event.Output.MessageOutput

		if msgOutput.IsStreaming {
			for {
				chunk, err := msgOutput.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					logger.Error("流式读取失败", logger.C(err))
					return err
				}
				if chunk == nil {
					continue
				}
				lastMsg.WriteString(chunk.Content)
				if err := callback(chunk.Content); err != nil {
					return err
				}
			}
		} else {
			message := msgOutput.Message
			if message == nil {
				continue
			}
			lastMsg.WriteString(message.Content)
			if err := callback(message.Content); err != nil {
				return err
			}
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
