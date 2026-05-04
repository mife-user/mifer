package bootstrap

import "mifer/pkg/conf"

// loadConfig 加载配置
func (a *Application) loadConfig() error {
	config, err := conf.LoadConfig()
	if err != nil {
		return err
	}
	if err := conf.StatusConfig(); err != nil {
		return err
	}
	a.Config = config
	return nil
}
