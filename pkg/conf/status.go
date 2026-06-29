package conf

import "mifer/pkg/errorer"

// 检查配置
func StatusConfig() error {
	//jwt配置检查
	if globalConfig.JWT.Secret == "" {
		return errorer.New(errorer.ErrJWTKeyNotConfigured)
	}
	//ai配置检查
	defaultBackend, ok := globalConfig.Ai.Backends["default"]
	if !ok || defaultBackend.Provider == "" {
		return errorer.New(errorer.ErrAIDefaultBackendNotConfigured)
	}
	if defaultBackend.BaseURL == "" && defaultBackend.Provider == "openai" {
		return errorer.New(errorer.ErrAIBaseURLNotConfigured)
	}
	if defaultBackend.Model == "" {
		return errorer.New(errorer.ErrAIModelNotConfigured)
	}
	// api_key 为空时不再阻止启动，运行时通过 /api/admin/status 提示用户配置
	return nil
}
