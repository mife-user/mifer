package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mifer/pkg/conf"
	"mifer/pkg/logger"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// createHabitGraph 创建用户习惯总结图：ChatModel(haiku) → Lambda(写入 MIFER.md)
// 图结构：START → habit_chat → habit_writer → END
// 返回 nil 表示图创建失败（仅记录日志，不中断 Init 流程）。
func createHabitGraph(ctx context.Context, chatModel model.BaseChatModel) compose.Runnable[[]*schema.Message, string] {
	g := compose.NewGraph[[]*schema.Message, string]()

	if err := g.AddChatModelNode("habit_chat", chatModel); err != nil {
		logger.Warn(ctx, "创建习惯总结图 ChatModel 节点失败", logger.C(err))
		return nil
	}
	if err := g.AddLambdaNode("habit_writer", compose.InvokableLambda(
		func(ctx context.Context, msg *schema.Message) (string, error) {
			miferPath := filepath.Join(conf.GetConfig().Path.CfgPath, "MIFER.md")
			if err := os.WriteFile(miferPath, []byte(msg.Content), 0644); err != nil {
				return "", fmt.Errorf("写入 MIFER.md 失败: %w", err)
			}
			logger.Debug(ctx, "用户画像已更新", logger.I("len", len(msg.Content)))
			return msg.Content, nil
		},
	)); err != nil {
		logger.Warn(ctx, "创建习惯总结图 Lambda 节点失败", logger.C(err))
		return nil
	}
	if err := g.AddEdge(compose.START, "habit_chat"); err != nil {
		logger.Warn(ctx, "创建习惯总结图边失败", logger.C(err))
		return nil
	}
	if err := g.AddEdge("habit_chat", "habit_writer"); err != nil {
		logger.Warn(ctx, "创建习惯总结图边失败", logger.C(err))
		return nil
	}
	if err := g.AddEdge("habit_writer", compose.END); err != nil {
		logger.Warn(ctx, "创建习惯总结图边失败", logger.C(err))
		return nil
	}

	compiled, err := g.Compile(ctx)
	if err != nil {
		logger.Warn(ctx, "编译习惯总结图失败", logger.C(err))
		return nil
	}
	return compiled
}
