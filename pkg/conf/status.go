package conf

import "fmt"

// 检查配置
func StatusConfig() error {
	//jwt配置检查
	if globalConfig.JWT.Secret == "" {
		return fmt.Errorf("jwt密钥未配置")
	}
	//ai配置检查
	if globalConfig.Ai.BaseURL == "" {
		return fmt.Errorf("ai基础URL未配置")
	}
	if globalConfig.Ai.Model == "" {
		return fmt.Errorf("ai模型未配置")
	}
	if globalConfig.Ai.ApiKey == "" {
		return fmt.Errorf("ai API密钥未配置")
	}
	return nil
}
