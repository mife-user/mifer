package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mifer/pkg/conf"
	"mifer/pkg/logger"
)

// NewManager 创建技能管理器，扫描指定目录加载所有技能
func NewManager(cfg conf.SkillConfig) (*Manager, error) {
	m := &Manager{
		cfg:    cfg,
		skills: make(map[string]*Skill),
	}

	// 解析技能目录路径
	m.skillsDir = cfg.Path
	if m.skillsDir == "" {
		wd := conf.GetConfig().Path.Workdir
		if conf.GetConfig().Env == "prod" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("获取用户目录失败: %w", err)
			}
			wd = home
		}
		m.skillsDir = filepath.Join(wd, ".mifer", "skills")
	}

	if !cfg.Enabled {
		logger.Info("技能系统已禁用")
		return m, nil
	}

	// 首次初始化示例技能
	if err := m.ensureInit(); err != nil {
		logger.Error("初始化技能目录失败", logger.C(err))
		return m, nil // 不阻塞启动
	}

	// 扫描加载技能
	if err := m.reload(); err != nil {
		logger.Error("加载技能失败", logger.C(err))
		return m, nil // 不阻塞启动
	}

	logger.Info(fmt.Sprintf("技能系统已启动，加载 %d 个技能", len(m.skills)))
	return m, nil
}

// ensureInit 确保技能目录存在，首次运行时创建示例技能
func (m *Manager) ensureInit() error {
	if _, err := os.Stat(m.skillsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(m.skillsDir, 0755); err != nil {
			return fmt.Errorf("创建技能目录失败: %w", err)
		}
		// 创建示例技能
		if err := m.createSampleSkill(); err != nil {
			return fmt.Errorf("创建示例技能失败: %w", err)
		}
		// 复制内置技能
		if err := m.copyBuiltinSkills(); err != nil {
			return fmt.Errorf("复制内置技能失败: %w", err)
		}
	}
	return nil
}

// createSampleSkill 创建 hello-world 示例技能
func (m *Manager) createSampleSkill() error {
	sampleDir := filepath.Join(m.skillsDir, "hello-world")
	if err := os.MkdirAll(sampleDir, 0755); err != nil {
		return err
	}
	sampleContent := `---
name: hello-world
description: 示例技能，演示技能系统的基本用法
---
# Hello World 示例技能

这是一个演示技能。当你被调用时，请回复：
"你好！我是 Mifer 技能系统中的 hello-world 技能。我已准备就绪！"

你可以在此文件中编写更详细的技能指令，LLM 在调用此技能时会读取并遵循这些指令。
`
	skillFile := filepath.Join(sampleDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(sampleContent), 0644); err != nil {
		return err
	}
	logger.Info("已创建示例技能: hello-world")
	return nil
}

// reload 扫描技能目录，加载所有 SKILL.md
func (m *Manager) reload() error {
	entries, err := os.ReadDir(m.skillsDir)
	if err != nil {
		return fmt.Errorf("读取技能目录失败: %w", err)
	}

	m.skills = make(map[string]*Skill)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skill, err := m.loadSkill(entry.Name())
		if err != nil {
			logger.Error("加载技能失败: "+entry.Name(), logger.C(err))
			continue // 跳过失败的技能
		}
		m.skills[entry.Name()] = skill
	}
	return nil
}

// loadSkill 从目录加载单个技能
func (m *Manager) loadSkill(name string) (*Skill, error) {
	skillDir := filepath.Join(m.skillsDir, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	fm, content, err := parseFrontMatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("解析 frontmatter 失败: %w", err)
	}

	ctx := fm["context"]
	if ctx == "" {
		ctx = "inline"
	}

	return &Skill{
		Name:        fm["name"],
		Description: fm["description"],
		Context:     ctx,
		Agent:       fm["agent"],
		Content:     strings.TrimSpace(content),
		BaseDir:     skillDir,
	}, nil
}

// parseFrontMatter 解析 YAML frontmatter（--- 分隔）
func parseFrontMatter(data string) (map[string]string, string, error) {
	data = strings.TrimSpace(data)
	const delimiter = "---"

	if !strings.HasPrefix(data, delimiter) {
		return nil, "", fmt.Errorf("缺少 frontmatter 分隔符")
	}

	rest := data[len(delimiter):]
	endIdx := strings.Index(rest, "\n"+delimiter)
	if endIdx == -1 {
		// 尝试只有 "---" 没有前导换行符
		endIdx = strings.Index(rest, delimiter)
		if endIdx == -1 {
			return nil, "", fmt.Errorf("缺少 frontmatter 结束分隔符")
		}
	}

	fmText := strings.TrimSpace(rest[:endIdx])
	content := rest[endIdx+len(delimiter):]
	if idx := strings.Index(content, "\n"); idx != -1 {
		content = content[idx+1:]
	}

	// 简单的手动解析（避免复杂依赖）
	fm := make(map[string]string)
	lines := strings.Split(fmText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			fm[key] = val
		}
	}

	return fm, content, nil
}

// List 返回所有已加载技能的简要信息
func (m *Manager) List() []SkillInfo {
	var infos []SkillInfo
	for _, s := range m.skills {
		infos = append(infos, SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			Context:     s.Context,
			Agent:       s.Agent,
		})
	}
	return infos
}

// Get 按名称获取技能
func (m *Manager) Get(name string) (*Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return nil, fmt.Errorf("技能 [%s] 不存在", name)
	}
	return s, nil
}

// SkillsDir 返回技能目录路径
func (m *Manager) SkillsDir() string {
	return m.skillsDir
}

// Count 返回已加载技能数量
func (m *Manager) Count() int {
	return len(m.skills)
}

// isEmpty 检查技能目录是否为空
func (m *Manager) isEmpty() bool {
	return len(m.skills) == 0
}
