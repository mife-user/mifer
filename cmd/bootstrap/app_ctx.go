package bootstrap

import (
	"context"
	"mifer/pkg/utils"
)

func (a *Application) initontext() error {
	id := []byte(a.Config.Workdir)
	idstr := utils.PseudoRandom(id)
	a.Context = context.WithValue(context.Background(), "id", idstr)
	return nil
}
