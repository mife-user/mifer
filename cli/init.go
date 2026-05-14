package cli

import (
	"bufio"
	"fmt"
	"mifer/cli/client"
	"mifer/pkg/conf"
	"os"
)

// Cli 命令行交互客户端
type Cli struct {
	client  *client.Client
	scanner *bufio.Scanner
}

// New 创建CLI实例
func New(config *conf.Config) *Cli {
	baseURL := fmt.Sprintf("http://localhost:%d", config.Gin.Port)
	return &Cli{
		client:  client.New(baseURL),
		scanner: bufio.NewScanner(os.Stdin),
	}
}
