package bootstrap

import "mifer/cli"

func (a *Application) initCli() error {
	a.Clier = cli.New(a.Config.Gin.Port)
	return nil
}
