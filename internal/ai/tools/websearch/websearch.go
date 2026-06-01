package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// WebSearchInput 网页搜索输入
type WebSearchInput struct {
	Query      string `json:"query" jsonschema:"required,description=搜索查询字符串"`
	MaxResults int    `json:"max_results" jsonschema:"description=返回结果数量，默认5，上限10"`
}

// SearchResultItem 单条搜索结果
type SearchResultItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchOutput 网页搜索输出
type WebSearchOutput struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
	Count   int                `json:"count"`
	Error   string             `json:"error,omitempty"`
}

// New 创建网页搜索工具
func New() (tool.InvokableTool, error) {
	return utils.InferTool("web_search",
		"搜索互联网获取最新信息。默认使用 SearXNG 元搜索引擎（聚合 Google/Bing/百度等多家结果），"+
			"也可配置为 Bing API 获取更高质量结果。",
		webSearch)
}

// webSearch 执行网页搜索
func webSearch(ctx context.Context, input WebSearchInput) (WebSearchOutput, error) {
	cfg := conf.GetConfig().Search

	if input.Query == "" {
		return WebSearchOutput{Error: "搜索查询不能为空"}, nil
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = cfg.MaxResults
		if maxResults <= 0 {
			maxResults = 5
		}
	}
	if maxResults > 10 {
		maxResults = 10
	}

	timeoutSec := cfg.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 10
	}

	switch cfg.Provider {
	case "bing":
		return bingAPISearch(ctx, input.Query, maxResults, timeoutSec, cfg)
	case "duckduckgo":
		return duckduckgoSearch(ctx, input.Query, maxResults, timeoutSec)
	default:
		return searxngSearch(ctx, input.Query, maxResults, timeoutSec, cfg)
	}
}

// searxngSearch 使用 SearXNG JSON API 搜索（免费，无限制，聚合多引擎结果）
func searxngSearch(ctx context.Context, query string, maxResults int, timeoutSec int, cfg conf.SearchConfig) (WebSearchOutput, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	reqURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=general&language=zh-CN",
		baseURL, neturl.QueryEscape(query))

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("创建 SearXNG 请求失败: %v", err)}, nil
	}
	req.Header.Set("User-Agent", "Mifer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("SearXNG 搜索请求失败（请确认 SearXNG 服务已启动）: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return WebSearchOutput{Error: fmt.Sprintf("SearXNG 返回错误状态码 %d: %s", resp.StatusCode, string(body))}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("读取 SearXNG 响应失败: %v", err)}, nil
	}

	var searxResp struct {
		Query   string `json:"query"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Engine  string `json:"engine"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &searxResp); err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("解析 SearXNG 响应失败: %v", err)}, nil
	}

	var results []SearchResultItem
	for _, item := range searxResp.Results {
		if len(results) >= maxResults {
			break
		}
		if item.Title == "" || item.URL == "" {
			continue
		}
		results = append(results, SearchResultItem{
			Title:   strings.TrimSpace(item.Title),
			URL:     item.URL,
			Snippet: strings.TrimSpace(item.Content),
		})
	}

	if len(results) == 0 {
		logger.Warn("SearXNG 搜索未返回结果", logger.S("query", query))
	}

	return WebSearchOutput{
		Query:   query,
		Results: results,
		Count:   len(results),
	}, nil
}

// bingAPISearch 使用 Bing Web Search API v7 搜索（需 Azure API Key，结果质量最高）
func bingAPISearch(ctx context.Context, query string, maxResults int, timeoutSec int, cfg conf.SearchConfig) (WebSearchOutput, error) {
	if cfg.APIKey == "" {
		return WebSearchOutput{Error: "Bing API 搜索需要配置 API 密钥（search.api_key），请注册 Azure 免费层获取"}, nil
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.bing.microsoft.com/v7.0/search"
	}

	reqURL := fmt.Sprintf("%s?q=%s&count=%d&mkt=zh-CN&textFormat=Raw",
		baseURL, neturl.QueryEscape(query), maxResults)

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("创建 Bing 请求失败: %v", err)}, nil
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", cfg.APIKey)
	req.Header.Set("User-Agent", "Mifer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("Bing API 请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return WebSearchOutput{Error: fmt.Sprintf("Bing API 返回错误状态码 %d: %s", resp.StatusCode, string(body))}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("读取 Bing API 响应失败: %v", err)}, nil
	}

	var bingResp struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.Unmarshal(body, &bingResp); err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("解析 Bing API 响应失败: %v", err)}, nil
	}

	var results []SearchResultItem
	for _, item := range bingResp.WebPages.Value {
		results = append(results, SearchResultItem{
			Title:   item.Name,
			URL:     item.URL,
			Snippet: item.Snippet,
		})
	}

	if len(results) == 0 {
		logger.Warn("Bing API 搜索未返回结果", logger.S("query", query))
	}

	return WebSearchOutput{
		Query:   query,
		Results: results,
		Count:   len(results),
	}, nil
}

// duckduckgoSearch 使用 DuckDuckGo API 搜索（免费，但国内可能被墙）
func duckduckgoSearch(ctx context.Context, query string, maxResults int, timeoutSec int) (WebSearchOutput, error) {
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&no_redirect=1",
		neturl.QueryEscape(query))

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("创建请求失败: %v", err)}, nil
	}
	req.Header.Set("User-Agent", "Mifer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("DuckDuckGo 搜索请求失败（国内可能需要代理）: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WebSearchOutput{Error: fmt.Sprintf("DuckDuckGo API 返回错误状态码: %d", resp.StatusCode)}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("读取搜索响应失败: %v", err)}, nil
	}

	var ddgResp struct {
		Abstract       string `json:"Abstract"`
		AbstractText   string `json:"AbstractText"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		Heading        string `json:"Heading"`
		RelatedTopics  []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return WebSearchOutput{Error: fmt.Sprintf("解析搜索响应失败: %v", err)}, nil
	}

	var results []SearchResultItem

	if ddgResp.AbstractText != "" {
		title := ddgResp.AbstractSource
		if title == "" {
			title = ddgResp.Heading
		}
		if title == "" {
			title = query
		}
		results = append(results, SearchResultItem{
			Title:   title,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.AbstractText,
		})
	}

	for _, topic := range ddgResp.RelatedTopics {
		if len(results) >= maxResults {
			break
		}
		if topic.Text == "" || topic.FirstURL == "" {
			continue
		}
		title, snippet := splitDDGText(topic.Text)
		results = append(results, SearchResultItem{
			Title:   title,
			URL:     topic.FirstURL,
			Snippet: snippet,
		})
	}

	for _, r := range ddgResp.Results {
		if len(results) >= maxResults {
			break
		}
		if r.Text == "" || r.FirstURL == "" {
			continue
		}
		title, snippet := splitDDGText(r.Text)
		results = append(results, SearchResultItem{
			Title:   title,
			URL:     r.FirstURL,
			Snippet: snippet,
		})
	}

	if len(results) == 0 {
		logger.Warn("DuckDuckGo 搜索未返回有效结果", logger.S("query", query))
	}

	return WebSearchOutput{
		Query:   query,
		Results: results,
		Count:   len(results),
	}, nil
}

// splitDDGText 分割 DuckDuckGo 的 "Title - Description" 格式文本
func splitDDGText(text string) (title, snippet string) {
	for i := 0; i < len(text)-2; i++ {
		if text[i] == ' ' && text[i+1] == '-' && text[i+2] == ' ' {
			return compressSpaces(text[:i]), compressSpaces(text[i+3:])
		}
	}
	return compressSpaces(text), compressSpaces(text)
}

// compressSpaces 压缩连续空白字符
func compressSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
