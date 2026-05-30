package mcp

import (
	"context"
	"encoding/json"

	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPToolAdapter 将 MCP Tool 适配为 Eino 的 tool.InvokableTool 接口
type MCPToolAdapter struct {
	mcpTool    mcp.Tool         // MCP 工具元数据
	mcpClient  client.MCPClient // MCP 客户端（用于执行 CallTool）
	serverName string           // 来源 Server 名
	fullName   string           // 完整工具名：{serverName}_{toolName}
}

// NewMCPToolAdapter 创建 MCP 工具适配器
func NewMCPToolAdapter(mcpTool mcp.Tool, mcpClient client.MCPClient, serverName string) *MCPToolAdapter {
	return &MCPToolAdapter{
		mcpTool:    mcpTool,
		mcpClient:  mcpClient,
		serverName: serverName,
		fullName:   serverName + "_" + mcpTool.Name,
	}
}

// Info 返回 Eino ToolInfo，将 MCP 的 JSON Schema 转换为 Eino 格式
func (a *MCPToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info := &schema.ToolInfo{
		Name: a.fullName,
		Desc: a.mcpTool.Description,
	}

	// 如果 MCP 工具有输入参数，转换 Schema
	if a.mcpTool.InputSchema.Properties != nil || len(a.mcpTool.InputSchema.Required) > 0 || a.mcpTool.InputSchema.Type == "object" {
		// 通过 JSON 序列化/反序列化将 MCP Schema 转为 Eino jsonschema.Schema
		rawJSON, err := json.Marshal(a.mcpTool.InputSchema)
		if err != nil {
			logger.Error("MCP工具 Schema 序列化失败: "+a.fullName, logger.C(err))
			return info, nil
		}

		var einoSchema jsonschema.Schema
		if err := json.Unmarshal(rawJSON, &einoSchema); err != nil {
			logger.Error("MCP工具 Schema 转换失败: "+a.fullName, logger.C(err))
			return info, nil
		}

		info.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(&einoSchema)
	}

	return info, nil
}

// InvokableRun 执行 MCP 工具，将结果返回给 LLM
func (a *MCPToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	var args map[string]any
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			logger.Error("MCP工具参数解析失败: "+a.fullName, logger.C(err))
			return "参数解析失败: " + err.Error(), nil
		}
	}

	// 构建 MCP CallToolRequest
	req := mcp.CallToolRequest{}
	req.Params.Name = a.mcpTool.Name
	req.Params.Arguments = args

	// 调用 MCP Server
	result, err := a.mcpClient.CallTool(ctx, req)
	if err != nil {
		logger.Error("MCP工具调用失败: "+a.fullName, logger.C(err))
		return "工具调用失败: " + err.Error(), nil
	}

	// 提取文本内容
	var texts []string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			texts = append(texts, textContent.Text)
		}
	}

	output := ""
	for i, t := range texts {
		if i > 0 {
			output += "\n"
		}
		output += t
	}

	// 如果工具返回了错误，将错误信息返回给 LLM（而非 Go error）
	if result.IsError {
		return "工具返回错误: " + output, nil
	}

	return output, nil
}

// FullName 返回完整工具名
func (a *MCPToolAdapter) FullName() string {
	return a.fullName
}
