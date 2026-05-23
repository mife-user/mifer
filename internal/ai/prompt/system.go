package prompt

// GetSystemPrompt 返回当前系统提示词
func (p *Prompty) GetSystemPrompt() string {
	return p.SystemPrompt
}

// SetSystemPrompt 设置自定义系统提示词
func (p *Prompty) SetSystemPrompt(prompt string) {
	p.SystemPrompt = prompt
}

// ResetSystemPrompt 重置为默认系统提示词
func (p *Prompty) ResetSystemPrompt() {
	p.SystemPrompt = defaultSystemPrompt
}
