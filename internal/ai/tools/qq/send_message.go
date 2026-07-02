package qqtools

import (
	"context"
	"fmt"

	"mifer/qq"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// NewSendMessage 创建 qq_send_message 工具，供 Agent 通过 Function Calling 发送 QQ 消息。
// getSender 延迟获取 Sender 实现，避免构造工具时 Sender 尚未初始化。
func NewSendMessage(getSender func() qq.Sender) (tool.InvokableTool, error) {
	return utils.InferTool("qq_send_message",
		"发送 QQ 消息到指定目标（私聊或群聊）。target_type 为 private 时发私聊，为 group 时发群聊。",
		func(ctx context.Context, input struct {
			TargetType string `json:"target_type" jsonschema:"required,description=目标类型 private(私聊) 或 group(群聊)"`
			TargetID   int64  `json:"target_id"   jsonschema:"required,description=目标ID 私聊为QQ号 群聊为群号"`
			Content    string `json:"content"     jsonschema:"required,description=要发送的消息内容"`
		}) (string, error) {
			sender := getSender()
			if sender == nil {
				return "", fmt.Errorf("QQ 消息服务未启用，请在配置中开启 qq.enabled")
			}
			switch input.TargetType {
			case "private":
				if err := sender.SendPrivateMsg(input.TargetID, input.Content); err != nil {
					return "", err
				}
			case "group":
				if err := sender.SendGroupMsg(input.TargetID, input.Content); err != nil {
					return "", err
				}
			default:
				return "", fmt.Errorf("未知目标类型 %s，可选 private 或 group", input.TargetType)
			}
			return "消息发送成功", nil
		},
	)
}
