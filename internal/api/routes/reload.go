package routes

import (
	"context"
	"fmt"
	"mifer/internal/ai/executor"
	"mifer/internal/api/dto/response/adminresp"
	"mifer/internal/service/agentservice"
	"mifer/internal/service/toolservice"
	"mifer/pkg/conf"
	"mifer/pkg/errorer"
	"mifer/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReloadHandler 处理配置重载请求
func (r *Router) ReloadHandler(c *gin.Context) {
	resp, err := r.Reload(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Reload 重新加载配置并重建模型与Agent树
func (r *Router) Reload(ctx context.Context) (*adminresp.ReloadResp, error) {
	if r.agentHandler == nil {
		return nil, errorer.New(errorer.ErrReloadHandlerNotReady)
	}

	// 快照当前配置，用于失败时回滚
	oldConfig := *conf.GetConfig()

	// 重新读取配置文件
	if _, err := conf.LoadConfig(); err != nil {
		logger.Error(ctx, "配置重载失败", logger.C(err))
		return nil, errorer.NewS(errorer.ErrConfigReloadFailed, err)
	}

	// 校验新配置
	if err := conf.StatusConfig(); err != nil {
		logger.Error(ctx, "新配置校验失败，回滚配置", logger.C(err))
		*conf.GetConfig() = oldConfig
		return nil, err
	}

	// 重建执行器
	newExec, err := executor.Init(r.appCtx)
	if err != nil {
		logger.Error(ctx, "重建执行器失败，回滚配置", logger.C(err))
		*conf.GetConfig() = oldConfig
		return nil, errorer.NewS(errorer.ErrRebuildExecutorFailed, err)
	}

	// 构建各后端状态报告
	resp := buildReloadResp(newExec)

	// 原子替换服务，并释放旧实例持有的资源（MCP 子进程、确认存储 actor）
	oldSvc := r.agentHandler.SwapService(agentservice.NewAgentService(newExec))
	if oldAgentSvc, ok := oldSvc.(*agentservice.AgentService); ok {
		oldAgentSvc.CloseExecutor()
	}

	// 原子替换工具服务
	oldToolSvc := r.toolHandler.SwapService(toolservice.NewToolService(newExec.Humen.ConfirmStore, conf.GetConfig().Path.Workdir))
	_ = oldToolSvc

	logger.Info(ctx, "配置重载成功", logger.S("backends", fmt.Sprintf("%v", newExec.Humen.Registry.Keys())))
	return resp, nil
}

// buildReloadResp 根据注册中心已加载的模型构建状态报告
func buildReloadResp(exec *executor.Executor) *adminresp.ReloadResp {
	resp := &adminresp.ReloadResp{Success: true, Message: "配置重载成功"}
	backends := conf.GetConfig().Ai.Backends
	loadedKeys := exec.Humen.Registry.Keys()
	loadedSet := make(map[string]bool, len(loadedKeys))
	for _, k := range loadedKeys {
		loadedSet[k] = true
	}

	for key, cfg := range backends {
		// embedding 类型后端由 RAG 管理，不属于聊天模型注册中心
		if cfg.Type == "embedding" {
			continue
		}
		status := adminresp.BackendStatus{Name: key, Model: cfg.Model}
		if loadedSet[key] {
			status.Status = "ok"
		} else if cfg.APIKey == "" {
			status.Status = "failed"
			status.Error = "api_key 未配置，请在 /config 中设置后重载"
		} else {
			status.Status = "failed"
			status.Error = "后端初始化失败，请检查provider、base_url、api_key配置"
		}
		resp.Backends = append(resp.Backends, status)
	}
	return resp
}
