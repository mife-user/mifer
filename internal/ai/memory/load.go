package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"os"

	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/schema"
)

// load 从 JSONL 文件逐行加载记忆数据，文件不存在时返回空列表
func load(cfg *MemCfg) ([]*schema.Message, error) {
	if err := validateID(cfg.Id); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cfg.MemPath, 0755); err != nil {
		logger.Error(context.Background(), "创建记忆目录失败", logger.C(err))
		return nil, errorer.New(errorer.ErrPathCannotCreate)
	}

	fileName, err := buildFilePath(cfg.MemPath, cfg.Id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		logger.Error(context.Background(), "打开记忆文件失败", logger.C(err))
		return nil, errorer.New(errorer.ErrArgUnknowid)
	}
	defer f.Close()

	var messages []*schema.Message
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg schema.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			logger.Error(context.Background(), "解析记忆行失败", logger.C(err))
			return nil, errorer.NewS(errorer.ErrParseLineFailed, err)
		}
		messages = append(messages, &msg)
	}
	if err := scanner.Err(); err != nil {
		logger.Error(context.Background(), "读取记忆文件失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrReadFileFailed, err)
	}
	return messages, nil
}
