package memory

import (
	"encoding/json"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"os"
)

// Save 将未持久化的新消息追加写入 JSONL 文件（每行一条 JSON）
func (m *Memory) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := validateID(m.Cfg.Id); err != nil {
		return err
	}
	if err := os.MkdirAll(m.Cfg.MemPath, 0755); err != nil {
		logger.Error("创建记忆目录失败", logger.C(err))
		return errorer.NewS(errorer.ErrCreateMemoryDirFailed, err)
	}

	newMsgs := m.messages[m.savedCount:]
	if len(newMsgs) == 0 {
		return nil
	}

	fileName, err := buildFilePath(m.Cfg.MemPath, m.Cfg.Id)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("打开记忆文件失败", logger.C(err))
		return errorer.NewS(errorer.ErrOpenFileFailed, err)
	}
	defer f.Close()

	for _, msg := range newMsgs {
		line, err := json.Marshal(msg)
		if err != nil {
			logger.Error("序列化记忆失败", logger.C(err))
			return errorer.NewS(errorer.ErrSerializeJSONFailed, err)
		}
		if _, err := f.Write(line); err != nil {
			logger.Error("写入记忆失败", logger.C(err))
			return errorer.NewS(errorer.ErrWriteFileFailed, err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			logger.Error("写入记忆换行符失败", logger.C(err))
			return errorer.NewS(errorer.ErrWriteNewlineFailed, err)
		}
	}
	m.savedCount = len(m.messages)
	return nil
}
