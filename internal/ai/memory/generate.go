package memory

import (
	"context"

	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"mifer/pkg/utils"
)

// GenerateID 使用与启动时相同的算法生成一个新的记忆会话ID
// 算法：utils.RandomStr(3) + Workdir → utils.PseudoRandom → 十进制字符串
func (m *Memory) GenerateID() (string, error) {
	random, err := utils.RandomStr(3)
	if err != nil {
		logger.Error(context.Background(), "生成随机ID失败", logger.C(err))
		return "", err
	}
	id := []byte(conf.GetConfig().Path.Workdir + random)
	return utils.PseudoRandom(id), nil
}
