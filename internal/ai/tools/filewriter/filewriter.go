package filewriter

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// FileWriterInput 文件写入工具输入参数
type FileWriterInput struct {
	FilePath  string `json:"file_path" jsonschema:"required,description=要写入的文件路径（绝对路径或相对路径）"`
	Content   string `json:"content" jsonschema:"required,description=要写入文件的内容"`
	Mode      string `json:"mode" jsonschema:"description=写入模式：write（覆盖写入）、append（追加写入）、insert（在start_line前插入）、replace_lines（替换start_line到end_line的行），默认write"`
	StartLine int    `json:"start_line" jsonschema:"description=insert/replace_lines模式的起始行号（1-based），仅insert和replace_lines模式使用"`
	EndLine   int    `json:"end_line" jsonschema:"description=replace_lines模式的结束行号（1-based，包含），仅replace_lines模式使用"`
}

// FileWriterOutput 文件写入工具输出结果
type FileWriterOutput struct {
	Success      bool   `json:"success"`
	BytesWritten int    `json:"bytes_written"`
	FilePath     string `json:"file_path"`
	LinesAffected int   `json:"lines_affected,omitempty"` // insert/replace_lines 模式影响的行数
	Error        string `json:"error,omitempty"`
}

func New() (tool.InvokableTool, error) {
	return utils.InferTool("file_writer", "安全写入内容到本地文件，支持覆盖、追加、行前插入和行范围替换四种模式，含路径安全校验。", writeFile)
}

func writeFile(_ context.Context, input FileWriterInput) (FileWriterOutput, error) {
	// 处理模式默认值
	mode := input.Mode
	if mode != "write" && mode != "append" && mode != "insert" && mode != "replace_lines" {
		mode = "write"
	}

	absPath, err := filepath.Abs(filepath.Clean(input.FilePath))
	if err != nil {
		return FileWriterOutput{Error: "路径解析失败: " + err.Error()}, nil
	}
	if strings.Contains(filepath.ToSlash(absPath), "..") {
		absPath, _ = filepath.Abs(filepath.Clean(strings.ReplaceAll(input.FilePath, "..", "")))
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return FileWriterOutput{Error: "创建目录失败: " + err.Error()}, nil
	}

	// insert 和 replace_lines 模式：需要先读取全部行
	if mode == "insert" || mode == "replace_lines" {
		return writeWithLineControl(absPath, input, mode)
	}

	// write 和 append 模式：直接写入
	var flag int
	if mode == "append" {
		flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	}

	f, err := os.OpenFile(absPath, flag, 0644)
	if err != nil {
		return FileWriterOutput{Error: "打开文件失败: " + err.Error()}, nil
	}
	defer f.Close()

	n, err := f.WriteString(input.Content)
	if err != nil {
		return FileWriterOutput{Error: "写入文件失败: " + err.Error()}, nil
	}

	return FileWriterOutput{
		Success:      true,
		BytesWritten: n,
		FilePath:     absPath,
	}, nil
}

// writeWithLineControl 处理 insert 和 replace_lines 模式的行级写入
func writeWithLineControl(absPath string, input FileWriterInput, mode string) (FileWriterOutput, error) {
	// 读取现有文件的所有行
	lines, err := readAllLines(absPath)
	if err != nil {
		return FileWriterOutput{Error: "读取文件失败: " + err.Error()}, nil
	}

	contentLines := strings.Split(input.Content, "\n")
	totalLines := len(lines)

	switch mode {
	case "insert":
		startLine := input.StartLine
		if startLine <= 0 {
			startLine = 1
		}
		if startLine > totalLines+1 {
			return FileWriterOutput{
				Error: fmt.Sprintf("插入行号%d超出范围，文件共%d行，允许的范围是1-%d", startLine, totalLines, totalLines+1),
			}, nil
		}

		// 在 startLine 之前插入（1-based），startLine=totalLines+1 时追加到末尾
		insertIdx := startLine - 1
		var newLines []string
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, contentLines...)
		newLines = append(newLines, lines[insertIdx:]...)
		lines = newLines

	case "replace_lines":
		startLine := input.StartLine
		endLine := input.EndLine
		if startLine <= 0 {
			startLine = 1
		}
		if endLine <= 0 {
			endLine = startLine
		}
		if startLine > totalLines {
			return FileWriterOutput{
				Error: fmt.Sprintf("替换起始行%d超出文件总行数%d", startLine, totalLines),
			}, nil
		}
		if endLine > totalLines {
			endLine = totalLines
		}
		if startLine > endLine {
			startLine, endLine = endLine, startLine
		}

		// 替换 startLine 到 endLine（含）之间的行
		var newLines []string
		newLines = append(newLines, lines[:startLine-1]...)
		newLines = append(newLines, contentLines...)
		newLines = append(newLines, lines[endLine:]...)
		lines = newLines
	}

	// 将结果写回文件
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return FileWriterOutput{Error: "写入文件失败: " + err.Error()}, nil
	}

	return FileWriterOutput{
		Success:       true,
		BytesWritten:  len(newContent),
		FilePath:      absPath,
		LinesAffected: len(contentLines),
	}, nil
}

// readAllLines 读取文件所有行
func readAllLines(absPath string) ([]string, error) {
	f, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
