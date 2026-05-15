package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"mifer/cli/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// RunTUI 运行 Bubble Tea TUI 交互模式
func (c *Cli) RunTUI() error {
	m := tui.NewModel(c.client, c.config)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// Run 运行CLI交互循环
func (c *Cli) Run() error {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║          Mifer CLI 终端              ║")
	fmt.Println("║  输入消息开始对话, exit 退出         ║")
	fmt.Println("║  /viewmemory  查看对话记忆           ║")
	fmt.Println("║  /excmem      交换记忆会话           ║")
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
		case "loadmemory", "/loadmemory":
			c.handleLoadMemory("")
		default:
			if strings.HasPrefix(input, "/viewmemory ") {
				id := strings.TrimSpace(strings.TrimPrefix(input, "/viewmemory "))
				c.handleLoadMemory(id)
			} else if strings.HasPrefix(input, "/excmem ") {
				id := strings.TrimSpace(strings.TrimPrefix(input, "/excmem "))
				c.handleExcmem(id)
			} else if strings.HasPrefix(input, "/") {
				fmt.Printf("未知命令: %s\n", input)
				c.printHelp()
			} else {
				c.handleChat(input)
			}
		}
	}

	return c.scanner.Err()
}

func (c *Cli) handleChat(content string) {
	ctx := context.Background()
	err := c.client.Chat.Send(ctx, content, func(event, chunk string) error {
		fmt.Print(chunk)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
	} else {
		fmt.Println()
	}
}

func (c *Cli) handleLoadMemory(id string) {
	if id == "" {
		id = "default"
	}
	memory, err := c.client.Memory.Load(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		return
	}
	if memory == "" {
		fmt.Println("(暂无对话记忆)")
	} else {
		fmt.Println("═══════════ 对话记忆 ═══════════")
		fmt.Print(memory)
		fmt.Println("═══════════════════════════════")
	}
}

func (c *Cli) handleExcmem(id string) {
	if err := c.client.Excmem.Exchange(id); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		return
	}
	fmt.Printf("已切换到记忆会话: %s\n", id)
}

// printHelp 打印帮助信息
func (c *Cli) printHelp() {
	fmt.Println("命令:")
	fmt.Println("  直接输入文本      与AI对话")
	fmt.Println("  /viewmemory       查看当前会话记忆")
	fmt.Println("  /viewmemory <id>  查看指定会话记忆")
	fmt.Println("  /excmem <id>      切换到指定记忆会话")
	fmt.Println("  exit/quit         退出")
	fmt.Println("  help              显示帮助")
}
