package bootstrap

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/utils"
)

// initontext 初始化应用上下文
func (a *Application) initontext(ctx context.Context) error {
	if ctx.Value("id") != nil {
		a.Context = ctx
		return nil
	}
	// 生成随机字符串（8 字节 = 64 bit 熵，足以防止会话 ID 碰撞）
	random, err := utils.RandomStr(8)
	if err != nil {
		return err
	}
	// 生成应用ID
	id := []byte(conf.GetConfig().Path.Workdir + random)
	idstr := utils.PseudoRandom(id)
	a.Context = context.WithValue(ctx, "id", idstr)
	return nil
}
