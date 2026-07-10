package executor

import (
	"context"
	"path/filepath"
	"sync"

	"mifer/internal/ai/agent"
	"mifer/internal/ai/compressor"
	"mifer/pkg/conf"
	"mifer/pkg/logger"
	"mifer/pkg/snapshot"

	"github.com/cloudwego/eino/adk"
)

// compressState 上下文压缩相关状态
type compressState struct {
	compressor       *compressor.Compressor // 上下文压缩器
	needsCompression bool                   // 下次对话前需要压缩标记
	lastPromptTokens int                    // 触发压缩时的 PromptTokens 快照
}

type Executor struct {
	Runner   *adk.Runner
	QQRunner *adk.Runner // QQ 通道专用 Runner（无工具 Agent），nil 时回退到 Runner
	Humen    *agent.Humen
	Token    *TokenUsage       // token 累计用量统计
	compress compressState     // 上下文压缩状态
	Snapshot *snapshot.Service // 文件快照服务
	chatMu   sync.Mutex        // 防止并发 Chat 调用导致 Memory 数据竞争
}

func Init(c context.Context) (*Executor, error) {
	// Tool 回调处理器改为 per-invocation 方式，在 Chat 中通过 adk.WithCallbacks 注入

	ag, err := agent.Init(c)
	if err != nil {
		logger.Error("初始化agent失败", logger.C(err))
		return nil, err
	}

	// 仅当 Agent 已成功创建时才初始化 Runner
	// api_key 未配置时 ag.Agent 为 nil，Runner 保持 nil，Chat 中将返回友好提示
	var runner *adk.Runner
	var qqRunner *adk.Runner
	if ag.Agent != nil {
		runner = adk.NewRunner(c, adk.RunnerConfig{
			Agent:           ag.Agent,
			EnableStreaming: true,
		})
	}
	if ag.QQAgent != nil {
		qqRunner = adk.NewRunner(c, adk.RunnerConfig{
			Agent:           ag.QQAgent,
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
		logger.Debug("初始化文件快照服务",
			logger.S("workdir", cfg.Path.Workdir),
			logger.S("baseDir", baseDir),
		)
		snapSvc = snapshot.New(cfg.Path.Workdir, baseDir)
		if err := snapSvc.InitBaseline(); err != nil {
			logger.Warn("初始化快照基线失败，禁用快照功能", logger.C(err))
			snapSvc = nil
		} else {
			logger.Info("文件快照服务初始化成功", logger.S("baseDir", baseDir))
		}
	} else {
		logger.Debug("文件快照功能未启用（snapshot_enabled=false）")
	}

	return &Executor{
		Runner:   runner,
		QQRunner: qqRunner,
		Humen:    ag,
		Token:    tokens,
		compress: compressState{
			compressor: compressor.NewCompressor(ag.Registry, ag.SkillManager),
		},
		Snapshot: snapSvc,
	}, nil
}
