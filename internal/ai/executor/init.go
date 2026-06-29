package executor

import (
	"context"
	"path/filepath"

	"mifer/internal/ai/agent"
	aicallback "mifer/internal/ai/callback"
	"mifer/internal/ai/compressor"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"mifer/pkg/snapshot"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
)

// compressState 上下文压缩相关状态
type compressState struct {
	compressor       *compressor.Compressor // 上下文压缩器
	needsCompression bool                   // 下次对话前需要压缩标记
	lastPromptTokens int                    // 触发压缩时的 PromptTokens 快照
}

type Executor struct {
	Runner   *adk.Runner
	Humen    *agent.Humen
	Token    *TokenUsage       // token 累计用量统计
	compress compressState     // 上下文压缩状态
	Snapshot *snapshot.Service // 文件快照服务
}

func Init(c context.Context) (*Executor, error) {
	// 注册全局 Tool 回调处理器，捕获所有 Tool 组件的 OnStart/OnEnd/OnError
	callbacks.AppendGlobalHandlers(aicallback.ToolCallbackHandler)

	ag, err := agent.Init(c)
	if err != nil {
		logger.Error("初始化agent失败", logger.C(err))
		return nil, err
	}

	// 仅当 Agent 已成功创建时才初始化 Runner
	// api_key 未配置时 ag.Agent 为 nil，Runner 保持 nil，Chat 中将返回友好提示
	var runner *adk.Runner
	if ag.Agent != nil {
		runner = adk.NewRunner(c, adk.RunnerConfig{
			Agent:           ag.Agent,
			EnableStreaming: true,
		})
	}

	// 初始化 token 用量统计
	tokens := initTokenUsage()

	// 初始化文件快照服务
	cfg := conf.GetConfig()
	var snapSvc *snapshot.Service
	if cfg.Path.SnapshotEnabled {
		memPath := filepath.Join(cfg.Path.CfgPath, "memory", filepath.Base(cfg.Path.Workdir))
		id, _ := c.Value("id").(string)
		baseDir := filepath.Join(memPath, id+"_snapshots")
		snapSvc = snapshot.New(cfg.Path.Workdir, baseDir)
		if err := snapSvc.InitBaseline(); err != nil {
			logger.Warn("初始化快照基线失败，禁用快照功能", logger.C(err))
			snapSvc = nil
		}
	}

	return &Executor{
		Runner: runner,
		Humen:  ag,
		Token:  tokens,
		compress: compressState{
			compressor: compressor.NewCompressor(ag.Registry, ag.SkillManager),
		},
		Snapshot: snapSvc,
	}, nil
}
