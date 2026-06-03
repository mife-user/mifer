package memory

import (
	"github.com/cloudwego/eino/schema"
)

// LoadByID 加载指定ID的记忆数据，仅读取磁盘文件，不修改当前活跃会话
func (m *Memory) LoadByID(id string) ([]*schema.Message, error) {
	cfg := MemCfg{MemPath: m.Cfg.MemPath, Id: id}
	msgs, err := load(&cfg)
	if err != nil {
		return nil, err
	}
	if msgs == nil {
		msgs = []*schema.Message{}
	}
	return msgs, nil
}
