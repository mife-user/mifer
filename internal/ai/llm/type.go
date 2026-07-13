package llm

import (
	"context"
	"mifer/pkg/conf"

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
	keysOrder []string // 维护注册顺序，用于 First()/FirstKey() 回退
}

// NewRegistry 创建一个新的 Registry，并注册所有内置的模型提供商
func NewRegistry() *Registry {
	r := &Registry{
		models:    make(map[string]model.BaseChatModel),
		providers: make(map[string]Provider),
		keysOrder: make([]string, 0),
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

// Get 按名称获取后端模型，不存在时返回 nil（调用方自行处理回退）
func (r *Registry) Get(name string) model.BaseChatModel {
	if m, ok := r.models[name]; ok {
		return m
	}
	return nil
}

// Has 检查指定名称的后端是否已加载
func (r *Registry) Has(name string) bool {
	_, ok := r.models[name]
	return ok
}

// IsReady 检查是否有任意后端已加载（即 AI 对话功能是否可用）
func (r *Registry) IsReady() bool {
	return len(r.models) > 0
}

// Keys 返回所有已注册的后端名称
func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.models))
	for k := range r.models {
		keys = append(keys, k)
	}
	return keys
}

// First 返回第一个注册的模型实例，用于回退
func (r *Registry) First() model.BaseChatModel {
	if len(r.keysOrder) == 0 {
		return nil
	}
	return r.models[r.keysOrder[0]]
}

// FirstKey 返回第一个注册的后端名称，用于回退时的名称获取
func (r *Registry) FirstKey() string {
	if len(r.keysOrder) == 0 {
		return ""
	}
	return r.keysOrder[0]
}

// registerModel 内部注册模型，记录顺序
func (r *Registry) registerModel(key string, m model.BaseChatModel) {
	r.models[key] = m
	r.keysOrder = append(r.keysOrder, key)
}
