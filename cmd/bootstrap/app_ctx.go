package bootstrap

import (
	"context"
	"mifer/pkg/utils"
)

// initontext 初始化应用上下文
func (a *Application) initontext(ctx context.Context) error {
	if ctx.Value("id") != nil {
		a.Context = ctx
		return nil
	}
	// 生成随机字符串
	random, err := utils.RandomStr(3)
	if err != nil {
		return err
	}
	// 生成应用ID
	id := []byte(a.Config.Path.Workdir + random)
	idstr := utils.PseudoRandom(id)
	a.Context = context.WithValue(ctx, "id", idstr)
	return nil
}
