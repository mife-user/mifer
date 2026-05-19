package llm

import (
	"github.com/cloudwego/eino/components/model"
)

// Registry 管理所有后端模型实例，按名称索引
type Registry struct {
	models map[string]model.BaseChatModel
}

// Get 按名称获取后端模型，不存在时 fallback 到 default
func (r *Registry) Get(name string) model.BaseChatModel {
	if m, ok := r.models[name]; ok {
		return m
	}
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
