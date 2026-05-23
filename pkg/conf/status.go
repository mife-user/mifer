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
	if defaultBackend.APIKey == "" {
		return errorer.New(errorer.ErrAIApiKeyNotConfigured)
	}
	return nil
}
