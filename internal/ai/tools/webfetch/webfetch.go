package webfetch

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
	"unicode"

	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"golang.org/x/net/html"
)

// WebFetchInput 网页抓取输入
type WebFetchInput struct {
	URL           string `json:"url" jsonschema:"required,description=要抓取的网页URL"`
	MaxContentLen int    `json:"max_content_len" jsonschema:"description=提取的文本内容最大长度（字符数），默认5000，上限20000。超过限制时在末尾截断"`
}

// WebFetchOutput 网页抓取输出
type WebFetchOutput struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	URL        string `json:"url"`
	ContentLen int    `json:"content_len"`
	Truncated  bool   `json:"truncated"`
	Error      string `json:"error,omitempty"`
}

// New 创建网页抓取工具
func New() (tool.InvokableTool, error) {
	return utils.InferTool("web_fetch",
		"抓取指定URL的网页内容并提取纯文本。用于获取和阅读网页文章、文档等在线内容。"+
			"自动过滤脚本、样式等非内容元素，返回结构化的标题和正文文本。",
		webFetch)
}

// webFetch 执行网页抓取
func webFetch(ctx context.Context, input WebFetchInput) (WebFetchOutput, error) {
	if input.URL == "" {
		return WebFetchOutput{Error: "URL 不能为空"}, nil
	}

	// 验证 URL 格式
	parsedURL, err := neturl.Parse(input.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return WebFetchOutput{Error: fmt.Sprintf("无效的 URL: %s（仅支持 http/https 协议）", input.URL)}, nil
	}

	// 防止 SSRF：拒绝内网地址
	if isPrivateHost(parsedURL.Hostname()) {
		return WebFetchOutput{Error: fmt.Sprintf("拒绝访问内网地址: %s", parsedURL.Hostname())}, nil
	}

	maxLen := input.MaxContentLen
	if maxLen <= 0 {
		maxLen = 5000
	}
	if maxLen > 20000 {
		maxLen = 20000
	}

	cfg := conf.GetConfig().Search
	timeoutSec := cfg.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 10
	}

	// 创建 HTTP 客户端，限制重定向
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("重定向次数过多")
			}
			// 防止 SSRF：拒绝重定向到内网地址
			if isPrivateHost(req.URL.Hostname()) {
				return fmt.Errorf("拒绝重定向到内网地址: %s", req.URL.Hostname())
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.URL, nil)
	if err != nil {
		return WebFetchOutput{Error: fmt.Sprintf("创建请求失败: %v", err)}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Mifer/1.0; +https://github.com/mifer)")

	resp, err := client.Do(req)
	if err != nil {
		return WebFetchOutput{Error: fmt.Sprintf("请求网页失败: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WebFetchOutput{Error: fmt.Sprintf("网页返回错误状态码: %d", resp.StatusCode)}, nil
	}

	// 检查 Content-Type（缺失时也拒绝，防止二进制内容被误解析）
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || !strings.Contains(strings.ToLower(contentType), "text/html") {
		return WebFetchOutput{Error: fmt.Sprintf("不支持的内容类型: %s（仅支持 text/html）", contentType)}, nil
	}

	// 限制读取大小（最大 2MB）
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return WebFetchOutput{Error: fmt.Sprintf("读取网页内容失败: %v", err)}, nil
	}

	// 解析 HTML 提取文本
	title, text := extractText(string(body))
	if text == "" {
		return WebFetchOutput{Error: "未能从网页中提取到有效文本内容"}, nil
	}

	truncated := false
	textRunes := []rune(text)
	if len(textRunes) > maxLen {
		text = string(textRunes[:maxLen])
		truncated = true
	}

	return WebFetchOutput{
		Title:      title,
		Content:    text,
		URL:        input.URL,
		ContentLen: len([]rune(text)),
		Truncated:  truncated,
	}, nil
}

// extractText 从 HTML 中提取标题和正文文本
func extractText(htmlContent string) (title string, text string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		logger.Warn(context.Background(), "解析 HTML 失败", logger.S("error", err.Error()))
		return "", ""
	}

	var titleBuilder strings.Builder
	var textBuilder strings.Builder
	inTitle := false
	inSkip := false
	skipDepth := 0

	// 要跳过的元素
	skipTags := map[string]bool{
		"script":   true,
		"style":    true,
		"noscript": true,
		"nav":      true,
		"footer":   true,
		"header":   true,
		"aside":    true,
	}

	var walker func(*html.Node)
	walker = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)

			if tag == "title" {
				inTitle = true
				titleBuilder.Reset()
			}

			if skipTags[tag] {
				inSkip = true
				skipDepth = 1
				// 不递归进入跳过元素
				goto skipChildren
			}
		}

		if n.Type == html.TextNode {
			content := strings.TrimSpace(n.Data)
			if content != "" {
				if inTitle {
					titleBuilder.WriteString(content)
					titleBuilder.WriteString(" ")
				}
				if !inSkip {
					textBuilder.WriteString(content)
					textBuilder.WriteString(" ")
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
			if inSkip {
				skipDepth--
				if skipDepth <= 0 {
					inSkip = false
				}
			}
		}

	skipChildren:
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "title" {
			inTitle = false
		}
		// 处理跳过元素的递归深度
		if inSkip && n.Type == html.ElementNode && !skipTags[strings.ToLower(n.Data)] {
			skipDepth++
		}
	}

	walker(doc)

	title = strings.TrimSpace(titleBuilder.String())
	text = collapseWhitespace(textBuilder.String())

	return title, text
}

// isPrivateHost 检查主机名是否为内网地址（防止 SSRF）。
func isPrivateHost(host string) bool {
	// 去除端口号
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// localhost / 回环地址
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	// 简单检查常见内网前缀
	if strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.16.") ||
		strings.HasPrefix(host, "192.168.") ||
		host == "169.254.169.254" || // AWS/云元数据
		host == "metadata.google.internal" {
		return true
	}
	return false
}

// collapseWhitespace 压缩连续空白字符
func collapseWhitespace(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))
	prevSpace := false

	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				builder.WriteRune(' ')
				prevSpace = true
			}
		} else {
			builder.WriteRune(r)
			prevSpace = false
		}
	}

	return strings.TrimSpace(builder.String())
}
