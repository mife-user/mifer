package agentconfig

import (
	"fmt"
	"slices"
)

// BuiltinNames 内置 Agent 名称列表，自定义 Agent 不可与其冲突
var BuiltinNames = []string{
	"Mifer", "MiEditer", "MiSummarizer", "MiPlanner", "MiCommander", "MiAuditor",
}

// CustomAgentConfig 代表 ~/.mifer/agents/<name>.yaml 中单个自定义 Agent 的配置
type CustomAgentConfig struct {
	Name          string   `yaml:"name"`           // 必填，唯一标识，不可与内置 Agent 重名
	Description   string   `yaml:"description"`    // 必填，Orchestrator 据此决定是否调度该 Agent
	Model         string   `yaml:"model"`          // 可选，默认 "default"，取值: default/sonnet/opus/haiku
	BaseDir       string   `yaml:"base_dir"`       // 可选，限制 file_writer/file_creator 的写入目录
	Instruction   string   `yaml:"instruction"`    // 必填，系统提示词
	Tools         []string `yaml:"tools"`          // 必填，至少一个工具名；支持 "mcp:<server>" 引用 MCP 工具
	MaxIterations int      `yaml:"max_iterations"` // 可选，0=默认20，正值=指定值，负值=无限
	Integration   string   `yaml:"integration"`    // 可选，默认 "orchestrator"，也可为 "standalone"
}

// Validate 校验必填字段和取值范围，返回首个错误或 nil
func (c *CustomAgentConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("Agent 名称不能为空")
	}
	if slices.Contains(BuiltinNames, c.Name) {
		return fmt.Errorf("Agent 名称 [%s] 与内置 Agent 冲突，保留名称为: %v", c.Name, BuiltinNames)
	}
	if c.Description == "" {
		return fmt.Errorf("Agent [%s] 描述不能为空", c.Name)
	}
	if c.Instruction == "" {
		return fmt.Errorf("Agent [%s] 系统提示词不能为空", c.Name)
	}
	if len(c.Tools) == 0 {
		return fmt.Errorf("Agent [%s] 至少需要一个工具", c.Name)
	}
	if c.Integration != "" && c.Integration != "orchestrator" && c.Integration != "standalone" {
		return fmt.Errorf("Agent [%s] integration 取值无效 [%s]，仅支持 orchestrator / standalone", c.Name, c.Integration)
	}
	if c.Model != "" && c.Model != "default" && c.Model != "sonnet" && c.Model != "opus" && c.Model != "haiku" && c.Model != "multi_modal" {
		return fmt.Errorf("Agent [%s] model 取值无效 [%s]，仅支持 default/sonnet/opus/haiku/multi_modal", c.Name, c.Model)
	}
	return nil
}

// GetMaxIterations 返回实际生效的最大迭代次数：0→20, 负值→0(无限), 正值→N
func (c *CustomAgentConfig) GetMaxIterations() int {
	if c.MaxIterations < 0 {
		return 0
	}
	if c.MaxIterations == 0 {
		return 20
	}
	return c.MaxIterations
}

// GetModel 返回模型名，空串回退为 "default"
func (c *CustomAgentConfig) GetModel() string {
	if c.Model == "" {
		return "default"
	}
	return c.Model
}

// IsOrchestrator 判断集成模式是否为 orchestrator（默认）
func (c *CustomAgentConfig) IsOrchestrator() bool {
	return c.Integration == "" || c.Integration == "orchestrator"
}
