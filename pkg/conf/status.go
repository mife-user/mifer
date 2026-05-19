package conf

import "fmt"

// 检查配置
func StatusConfig() error {
	//jwt配置检查
	if globalConfig.JWT.Secret == "" {
		return fmt.Errorf("jwt密钥未配置")
	}
	//ai配置检查
	defaultBackend, ok := globalConfig.Ai.Backends["default"]
	if !ok || defaultBackend.Provider == "" {
		return fmt.Errorf("ai默认后端未配置，请配置 ai.backends.default")
	}
	if defaultBackend.BaseURL == "" && defaultBackend.Provider == "openai" {
		return fmt.Errorf("ai默认后端 base_url 未配置")
	}
	if defaultBackend.Model == "" {
		return fmt.Errorf("ai默认后端模型未配置")
	}
	if defaultBackend.APIKey == "" {
		return fmt.Errorf("ai默认后端 api_key 未配置")
	}
	return nil
}
