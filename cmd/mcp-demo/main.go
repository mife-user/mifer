// Mifer MCP 演示工具 — 用于测试 MCP 客户端功能
// 编译: go build -o mcp-demo.exe ./cmd/mcp-demo
// MCP 配置中使用此可执行文件即可测试
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 创建 MCP Server
	s := server.NewMCPServer(
		"Mifer Demo Tools",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// ── echo: 回声工具 ──
	s.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("回声工具，将输入文本原样返回，用于测试 MCP 连接是否正常"),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("要回显的消息文本"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, _ := req.GetArguments()["message"].(string)
			if msg == "" {
				msg = "(空消息)"
			}
			return mcp.NewToolResultText(fmt.Sprintf("🔊 Echo: %s", msg)), nil
		},
	)

	// ── get_time: 获取当前时间 ──
	s.AddTool(
		mcp.NewTool("get_time",
			mcp.WithDescription("获取当前系统时间，支持指定时区"),
			mcp.WithString("timezone",
				mcp.Description("时区，如 Asia/Shanghai、America/New_York，留空使用本地时区"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tzStr, _ := req.GetArguments()["timezone"].(string)
			now := time.Now()
			location := now.Location()

			if tzStr != "" {
				loc, err := time.LoadLocation(tzStr)
				if err != nil {
					return mcp.NewToolResultText(fmt.Sprintf("❌ 无效时区: %s，请使用如 Asia/Shanghai 的格式", tzStr)), nil
				}
				location = loc
			}
			now = time.Now().In(location)
			return mcp.NewToolResultText(fmt.Sprintf("🕐 当前时间: %s (%s)", now.Format("2006-01-02 15:04:05"), location.String())), nil
		},
	)

	// ── calculator: 简单计算器 ──
	s.AddTool(
		mcp.NewTool("calculator",
			mcp.WithDescription("简单计算器，支持加减乘除四则运算"),
			mcp.WithString("operation",
				mcp.Required(),
				mcp.Description("运算类型: add(加)、subtract(减)、multiply(乘)、divide(除)"),
			),
			mcp.WithNumber("a",
				mcp.Required(),
				mcp.Description("第一个操作数"),
			),
			mcp.WithNumber("b",
				mcp.Required(),
				mcp.Description("第二个操作数"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			op, _ := req.GetArguments()["operation"].(string)
			a, _ := req.GetArguments()["a"].(float64)
			b, _ := req.GetArguments()["b"].(float64)

			var result float64
			var symbol string
			switch op {
			case "add":
				result = a + b
				symbol = "+"
			case "subtract":
				result = a - b
				symbol = "-"
			case "multiply":
				result = a * b
				symbol = "×"
			case "divide":
				if b == 0 {
					return mcp.NewToolResultText("❌ 除数不能为 0"), nil
				}
				result = a / b
				symbol = "÷"
			default:
				return mcp.NewToolResultText(fmt.Sprintf("❌ 不支持的运算: %s，请使用 add/subtract/multiply/divide", op)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("🧮 %.2f %s %.2f = %.4f", a, symbol, b, result)), nil
		},
	)

	// ── random_number: 随机数生成 ──
	s.AddTool(
		mcp.NewTool("random_number",
			mcp.WithDescription("生成指定范围内的随机整数"),
			mcp.WithNumber("min",
				mcp.Description("最小值（含），默认 1"),
			),
			mcp.WithNumber("max",
				mcp.Description("最大值（含），默认 100"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			minF, _ := req.GetArguments()["min"].(float64)
			maxF, _ := req.GetArguments()["max"].(float64)
			min := int(minF)
			max := int(maxF)
			if min == 0 && max == 0 {
				min, max = 1, 100
			}
			if min > max {
				min, max = max, min
			}
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			num := rng.Intn(max-min+1) + min
			return mcp.NewToolResultText(fmt.Sprintf("🎲 随机数 [%d, %d]: %d", min, max, num)), nil
		},
	)

	// 启动 stdio 服务
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("MCP Server 启动失败: %v\n", err)
	}
}
