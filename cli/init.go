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
	config  *conf.Config
}

// New 创建CLI实例
func New(config *conf.Config) *Cli {
	baseURL := fmt.Sprintf("http://%s:%d", config.Cli.Host, config.Cli.Port)
	return &Cli{
		client:  client.New(baseURL),
		scanner: bufio.NewScanner(os.Stdin),
		config:  config,
	}
}
