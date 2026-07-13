package conf

import "mifer/pkg/errorer"

// 检查配置
func StatusConfig() error {
	//jwt配置检查
	if globalConfig.JWT.Secret == "" {
		return errorer.New(errorer.ErrJWTKeyNotConfigured)
	}
	//ai配置检查 — 找到第一个 chat 类型后端进行校验
	firstChatBackend := ""
	var firstChatCfg BackendConfig
	for name, cfg := range globalConfig.Ai.Backends {
		if cfg.Type != "embedding" {
			firstChatBackend = name
			firstChatCfg = cfg
			break
		}
	}
	if firstChatBackend == "" || firstChatCfg.Provider == "" {
		return errorer.New(errorer.ErrNoBackendAvailable)
	}
	if firstChatCfg.BaseURL == "" && firstChatCfg.Provider == "openai" {
		return errorer.New(errorer.ErrAIBaseURLNotConfigured)
	}
	if firstChatCfg.Model == "" {
		return errorer.New(errorer.ErrAIModelNotConfigured)
	}
	// api_key 为空时不再阻止启动，运行时通过 /api/admin/status 提示用户配置
	return nil
}
