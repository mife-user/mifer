package bootstrap

import "mifer/cli"

func (a *Application) initCli() error {
	a.Clier = cli.New()
	return nil
}
