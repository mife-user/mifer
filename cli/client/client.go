package client

import (
	"net/http"

	"mifer/cli/client/chathandler"
	"mifer/cli/client/memhandler"
)

// Client HTTP API客户端，按服务聚合各Handler
type Client struct {
	Chat   *chathandler.ChatHandler
	Memory *memhandler.MemHandler
}

// New 创建API客户端实例
func New(baseURL string) *Client {
	httpClient := http.DefaultClient
	return &Client{
		Chat:   chathandler.New(httpClient, baseURL),
		Memory: memhandler.New(httpClient, baseURL),
	}
}
