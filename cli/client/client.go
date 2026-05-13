package client

import (
	"net/http"

	"mifer/cli/client/chathandler"
	"mifer/cli/client/excmemhandler"
	"mifer/cli/client/memhandler"
)

const (
	APIChatPath         = "/api/ai/chat"
	APIMemoryPath       = "/api/memory"
	APIExcmemPath       = "/api/memory/exchange"
)

// Client HTTP API客户端，按服务聚合各Handler
type Client struct {
	Chat    *chathandler.ChatHandler
	Memory  *memhandler.MemHandler
	Excmem  *excmemhandler.ExcmemHandler
}

// New 创建API客户端实例
func New(baseURL string) *Client {
	httpClient := http.DefaultClient
	return &Client{
		Chat:   chathandler.New(httpClient, baseURL+APIChatPath),
		Memory: memhandler.New(httpClient, baseURL+APIMemoryPath),
		Excmem: excmemhandler.New(httpClient, baseURL+APIExcmemPath),
	}
}
