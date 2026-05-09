package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Cli 命令行交互客户端
type Cli struct {
	baseURL string
	chatURL string
	scanner *bufio.Scanner
}

// New 创建CLI实例
func New(port int) *Cli {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	return &Cli{
		baseURL: baseURL,
		chatURL: baseURL + "/api/ai/chat",
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// Run 运行CLI交互循环
func (c *Cli) Run() error {
	fmt.Println("╔══════════════════════════════╗")
	fmt.Println("║       Mifer CLI 终端         ║")
	fmt.Println("║  输入消息开始对话, exit 退出  ║")
	fmt.Println("╚══════════════════════════════╝")
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
			fmt.Println("直接输入消息与AI对话, 输入 exit/quit 退出")
		default:
			if err := c.chat(input); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			}
		}
	}

	return c.scanner.Err()
}
