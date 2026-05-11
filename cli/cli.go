package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// apiEndpoints 集中管理所有API端点
type apiEndpoints struct {
	base   string
	chat   string
	memory string
}

// Cli 命令行交互客户端
type Cli struct {
	api     apiEndpoints
	scanner *bufio.Scanner
}

// New 创建CLI实例
func New(port int) *Cli {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	return &Cli{
		api: apiEndpoints{
			base:   baseURL,
			chat:   baseURL + "/api/ai/chat",
			memory: baseURL + "/api/memory",
		},
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// Run 运行CLI交互循环
func (c *Cli) Run() error {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║          Mifer CLI 终端              ║")
	fmt.Println("║  输入消息开始对话, exit 退出          ║")
	fmt.Println("║  /viewmemory 查看对话记忆            ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("再见!")
			return nil
		case "help":
			c.printHelp()
		case "viewmemory", "/viewmemory":
			if err := c.viewMemory(""); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			}
		default:
			// 检查是否是 /viewmemory 带参数
			if strings.HasPrefix(input, "/viewmemory ") {
				id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory "))
				if err := c.viewMemory(id); err != nil {
					fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				}
			} else if strings.HasPrefix(input, "/") {
				fmt.Printf("未知命令: %s\n", input)
				c.printHelp()
			} else {
				if err := c.chat(input); err != nil {
					fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				}
			}
		}
	}

	return c.scanner.Err()
}

// printHelp 打印帮助信息
func (c *Cli) printHelp() {
	fmt.Println("命令:")
	fmt.Println("  直接输入文本      与AI对话")
	fmt.Println("  /viewmemory       查看当前会话记忆")
	fmt.Println("  /viewmemory <id>  查看指定会话记忆")
	fmt.Println("  exit/quit         退出")
	fmt.Println("  help              显示帮助")
}
