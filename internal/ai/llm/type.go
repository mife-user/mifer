package llm

import (
	"context"
	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/model"
)

// Provider 定义了模型提供商的接口，每个提供商必须实现自己的名称标识和初始化逻辑
type Provider interface {
	// Name 返回提供商的名称标识（如 "openai"、"claude"、"gemini"、"ollama"）
	Name() string
	// InitModel 根据后端配置创建对应的 ChatModel 实例
	InitModel(ctx context.Context, cfg conf.BackendConfig) (model.BaseChatModel, error)
}

// Registry 管理所有后端模型实例和提供商注册，按名称索引
type Registry struct {
	models    map[string]model.BaseChatModel
	providers map[string]Provider
}

// NewRegistry 创建一个新的 Registry，并注册所有内置的模型提供商
func NewRegistry() *Registry {
	r := &Registry{
		models:    make(map[string]model.BaseChatModel),
		providers: make(map[string]Provider),
	}
	r.RegisterProvider(&openAIProvider{})
	r.RegisterProvider(&claudeProvider{})
	r.RegisterProvider(&geminiProvider{})
	r.RegisterProvider(&ollamaProvider{})
	return r
}

// RegisterProvider 向注册中心注册一个模型提供商
func (r *Registry) RegisterProvider(p Provider) {
	r.providers[p.Name()] = p
}

// Get 按名称获取后端模型，不存在时 fallback 到 default
func (r *Registry) Get(name string) model.BaseChatModel {
	if m, ok := r.models[name]; ok {
		return m
	}
	logger.Warn("模型后端未注册，降级到default", logger.S("requested", name))
	return r.models["default"]
}

// Keys 返回所有已注册的后端名称
func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.models))
	for k := range r.models {
		keys = append(keys, k)
	}
	return keys
}
