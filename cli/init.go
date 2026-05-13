package cli

import (
	"bufio"
	"fmt"
	"mifer/cli/client"
	"os"
)

// Cli 命令行交互客户端
type Cli struct {
	client  *client.Client
	scanner *bufio.Scanner
}

// New 创建CLI实例
func New(port int) *Cli {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	return &Cli{
		client:  client.New(baseURL),
		scanner: bufio.NewScanner(os.Stdin),
	}
}
