package commandexecutor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CommandExecutorInput 命令执行工具输入参数
type CommandExecutorInput struct {
	Command     string `json:"command" jsonschema:"required,description=要执行的shell命令"`
	WorkingDir  string `json:"working_dir" jsonschema:"description=命令执行的工作目录，默认为项目工作目录"`
	TimeoutSecs int    `json:"timeout_seconds" jsonschema:"description=命令超时时间（秒），默认30秒，最大120秒"`
}

// CommandExecutorOutput 命令执行工具输出结果
type CommandExecutorOutput struct {
	Success    bool   `json:"success"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
	Truncated  bool   `json:"truncated"`
	Error      string `json:"error,omitempty"`
}

const (
	defaultTimeout = 30
	maxTimeout     = 120
	maxOutputSize  = 100 * 1024 // 100KB
)

// 危险命令正则模式
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+-rf`),
	regexp.MustCompile(`rm\s+-r\s+-f`),
	regexp.MustCompile(`mkfs\.`),
	regexp.MustCompile(`dd\s+if=`),
	regexp.MustCompile(`chmod\s+.*777`),
	regexp.MustCompile(`sudo\s`),
	regexp.MustCompile(`curl.*\|\s*(ba)?sh`),
	regexp.MustCompile(`wget.*\|\s*(ba)?sh`),
	regexp.MustCompile(`>[^>]*\/etc\/`),
	regexp.MustCompile(`chown\s`),
	regexp.MustCompile(`:\(\)\s*\{`), // fork bomb
	regexp.MustCompile(`kill\s+-9`),
	regexp.MustCompile(`>\/dev\/sd`),
	regexp.MustCompile(`>\/dev\/hd`),
	regexp.MustCompile(`>\/dev\/nvme`),
	regexp.MustCompile(`>\/dev\/xvd`),
}

// 交互式/TTY命令检测
var interactivePattern = regexp.MustCompile(`(?i)^\s*(ssh|vim|vi|nano|top|htop|less|more|su|login|passwd|mysql|psql|telnet)\s`)

// 系统电源命令检测
var systemPowerPattern = regexp.MustCompile(`(?i)^\s*(reboot|shutdown|halt|poweroff|init\s+[06])\s*`)

// limitWriter 限制写入大小的Writer
type limitWriter struct {
	buf       bytes.Buffer
	maxSize   int
	truncated bool
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.truncated {
		return len(p), nil
	}
	remain := w.maxSize - w.buf.Len()
	if remain <= 0 {
		w.truncated = true
		return len(p), nil
	}
	if len(p) > remain {
		w.buf.Write(p[:remain])
		w.truncated = true
		return len(p), nil
	}
	return w.buf.Write(p)
}

func (w *limitWriter) String() string {
	result := w.buf.String()
	if w.truncated {
		result += fmt.Sprintf("\n... [输出已截断，超出%dKB限制]", w.maxSize/1024)
	}
	return result
}

func New() (tool.InvokableTool, error) {
	execute := func(ctx context.Context, input CommandExecutorInput) (CommandExecutorOutput, error) {
		return executeCommand(ctx, input)
	}
	return utils.InferTool("command_executor", "安全执行shell命令，包含危险命令检测、工作目录限制、超时控制和输出大小限制。", execute)
}

func executeCommand(ctx context.Context, input CommandExecutorInput) (CommandExecutorOutput, error) {
	cfg := conf.GetConfig()
	// 1. 校验命令非空
	if strings.TrimSpace(input.Command) == "" {
		return CommandExecutorOutput{Error: "命令不能为空"}, nil
	}

	// 2. 白名单检查（若配置了白名单）
	allowList, err := conf.LoadAllowList()
	if err != nil {
		logger.Warn("加载白名单失败", logger.C(err))
	}
	if len(allowList) > 0 {
		if !isAllowed(input.Command, allowList) {
			return CommandExecutorOutput{
				Error: "命令不在白名单中，已拒绝执行。允许的命令: " + strings.Join(allowList, ", "),
			}, nil
		}
	}

	// 3. 危险命令检测
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(input.Command) {
			logger.Warn("拦截危险命令", logger.S("command", input.Command), logger.S("pattern", pattern.String()))
			return CommandExecutorOutput{
				Error: "命令包含危险操作，已拒绝执行。匹配规则: " + pattern.String(),
			}, nil
		}
	}

	// 4. 系统电源命令检测
	if systemPowerPattern.MatchString(input.Command) {
		logger.Warn("拦截系统电源命令", logger.S("command", input.Command))
		return CommandExecutorOutput{
			Error: "禁止执行系统电源管理命令（reboot/shutdown/halt/poweroff/init），已拒绝执行",
		}, nil
	}

	// 5. 交互式命令检测
	if interactivePattern.MatchString(input.Command) {
		return CommandExecutorOutput{
			Error: "禁止执行需要交互终端的命令，已拒绝执行",
		}, nil
	}

	// 6. 工作目录沙箱
	workDir := input.WorkingDir
	if workDir == "" {
		workDir = cfg.Path.Workdir
	}
	absWorkDir, err := resolveSandboxDir(workDir, cfg.Path.Workdir)
	if err != nil {
		return CommandExecutorOutput{Error: "工作目录校验失败: " + err.Error()}, nil
	}

	// 7. 超时处理
	timeoutSecs := input.TimeoutSecs
	if timeoutSecs <= 0 {
		timeoutSecs = defaultTimeout
	}
	if timeoutSecs > maxTimeout {
		timeoutSecs = maxTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	// 8. 构建命令
	// Windows: 使用 PowerShell 而非 cmd.exe，因为 AI 倾向于生成 Unix 风格命令，
	// PowerShell 内置了大量 Unix 别名（ls/cat/curl/rm 等），兼容性远好于 cmd
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "powershell", "-Command", input.Command)
	} else {
		cmd = exec.CommandContext(execCtx, "bash", "-c", input.Command)
	}
	cmd.Dir = absWorkDir

	// 9. 捕获输出（带大小限制）
	stdoutWriter := &limitWriter{maxSize: maxOutputSize}
	stderrWriter := &limitWriter{maxSize: maxOutputSize}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	// 10. 执行命令
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// context超时或其他错误
			if execCtx.Err() == context.DeadlineExceeded {
				return CommandExecutorOutput{
					Command:    input.Command,
					WorkingDir: absWorkDir,
					Stdout:     stdoutWriter.String(),
					Stderr:     stderrWriter.String(),
					Error:      fmt.Sprintf("命令执行超时（%d秒）", timeoutSecs),
				}, nil
			}
			return CommandExecutorOutput{
				Command:    input.Command,
				WorkingDir: absWorkDir,
				Stdout:     stdoutWriter.String(),
				Stderr:     stderrWriter.String(),
				Error:      "命令执行失败: " + err.Error(),
			}, nil
		}
	}

	truncated := stdoutWriter.truncated || stderrWriter.truncated

	return CommandExecutorOutput{
		Success:    exitCode == 0,
		Stdout:     stdoutWriter.String(),
		Stderr:     stderrWriter.String(),
		ExitCode:   exitCode,
		Command:    input.Command,
		WorkingDir: absWorkDir,
		Truncated:  truncated,
	}, nil
}

// resolveSandboxDir 解析工作目录并校验是否在沙箱内
func resolveSandboxDir(workDir, projectDir string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(workDir))
	if err != nil {
		logger.Error("解析沙箱工作目录失败", logger.C(err))
		return "", err
	}
	absProject, err := filepath.Abs(filepath.Clean(projectDir))
	if err != nil {
		logger.Error("解析项目目录失败", logger.C(err))
		return "", err
	}
	// 规范化路径分隔符比较
	if !strings.HasPrefix(strings.ToLower(filepath.ToSlash(abs)), strings.ToLower(filepath.ToSlash(absProject))) {
		return "", errorer.NewF(errorer.ErrWorkDirNotInProject, absProject)
	}
	return abs, nil
}

// isAllowed 检查命令是否在白名单中（支持 * 通配符前缀匹配）
func isAllowed(command string, allowList []string) bool {
	trimmed := strings.TrimSpace(command)
	for _, allowed := range allowList {
		if prefix, ok := strings.CutSuffix(allowed, "*"); ok {
			if strings.HasPrefix(trimmed, prefix) {
				return true
			}
		} else if trimmed == allowed {
			return true
		}
	}
	return false
}
