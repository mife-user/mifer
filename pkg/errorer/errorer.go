package errorer

import (
	"errors"
	"fmt"
)

const (
	// 通用错误
	ErrChatTimeout  = "ctx killed or timeout"
	ErrCallBackNull = "AI 未生成回复内容"
	ErrArgUnknowid  = "未知的参数ID"
	ErrPathCannotCreate = "路径创建失败"
	ErrIdEmpty      = "ID 不能为空"

	// 后端配置
	ErrNoBackendConfig      = "未配置任何模型后端，请在配置中添加 ai.backends"
	ErrDefaultBackendConfig = "默认后端未配置，请在 ai.backends 中添加 default"
	ErrApiKey               = "apikey未配置"

	// Embedder
	ErrEmbedderBackendConfig = "未找到 embedder 后端配置，请在 backends 中配置 embedder"
	ErrEmbedderModelEmpty    = "embedder 模型名称为空"
	ErrCreateEmbedderFailed  = "创建Ollama嵌入器失败"

	// 向量存储
	ErrCreateIndexFailed     = "创建向量索引失败"
	ErrCreateRetrieverFailed = "创建Retriever失败"

	// RAG 服务
	ErrInitEmbedderFailed   = "初始化嵌入器失败"
	ErrInitFileLoaderFailed = "初始化文件加载器失败"
	ErrInitChunkerFailed    = "初始化分块器失败"
	ErrInitQdrantFailed     = "初始化Qdrant客户端失败"
	ErrInitIndexerFailed    = "初始化Indexer失败"
	ErrInitRetrieverFailed  = "初始化Retriever失败"
	ErrLoadFileFailed       = "加载文件失败"
	ErrChunkProcessFailed   = "分块处理失败"
	ErrVectorStoreFailed    = "向量存储失败"
	ErrVectorRetrieveFailed = "向量检索失败"

	// Chunker
	ErrCreateRecursiveChunkerFailed = "创建递归分块器失败"

	// RAG
	ErrRAGRetrieveFailed = "RAG检索失败"
	ErrRAGNotReady       = "知识库服务正在初始化，请稍后重试"

	// Prompt

	// Agent
	ErrIDNotString = "id is not string"

	// Memory
	ErrIDIllegalChars      = "id 包含非法字符: %s"
	ErrIndexOutOfRange     = "%s"
	ErrCreateMemoryDirFailed = "创建内存目录失败"
	ErrOpenFileFailed        = "打开文件失败"
	ErrSerializeJSONFailed   = "序列化JSON失败"
	ErrWriteFileFailed       = "写入文件失败"
	ErrWriteNewlineFailed    = "写入换行失败"
	ErrParseLineFailed       = "解析行失败"
	ErrReadFileFailed        = "读取文件失败"

	// CLI 客户端
	ErrSerializeRequestFailed = "序列化请求失败"
	ErrCreateRequestFailed    = "创建请求失败"
	ErrRequestFailed          = "请求失败"
	ErrReadResponseFailed     = "读取响应失败"
	ErrSSEScannerFailed       = "SSE扫描器错误"
	ErrParseResponseFailed    = "解析响应失败"
	ErrServerStatusCode       = "服务器返回状态码: %d"
	ErrServerStatusCodeDetail = "服务器返回状态码: %d, 响应: %s"
	ErrServerError            = "服务器错误: %s"

	// Bootstrap
	ErrCreateRouterFailed = "创建路由失败"
	ErrLoadConfigFailed   = "加载配置失败"
	ErrInitContextFailed  = "初始化上下文失败"
	ErrInitLoggerFailed   = "初始化日志失败"
	ErrInitRouterFailed   = "初始化路由失败"
	ErrInitCLIFailed      = "初始化CLI失败"
	ErrServerRunFailed    = "服务器运行失败"

	// Auth
	ErrTokenInvalid = "token无效"

	// 配置
	ErrGetWorkDirFailed              = "获取当前工作目录失败"
	ErrGetHomeDirFailed              = "获取用户主目录失败"
	ErrCreateDefaultConfigFailed     = "创建默认配置失败"
	ErrLoadMainConfigFailed          = "加载主配置失败"
	ErrParseConfigFailed             = "解析配置失败"
	ErrJWTKeyNotConfigured           = "jwt密钥未配置"
	ErrAIDefaultBackendNotConfigured = "ai默认后端未配置，请配置 ai.backends.default"
	ErrAIBaseURLNotConfigured        = "ai默认后端 base_url 未配置"
	ErrAIModelNotConfigured          = "ai默认后端模型未配置"
	ErrAIApiKeyNotConfigured         = "ai默认后端 api_key 未配置"
	ErrCreateConfigDirFailed         = "创建配置目录失败"
	ErrWriteDefaultConfigFailed      = "写入默认配置失败"

	// Logger
	ErrRotateFailed = "rotate failed"

	// LLM
	ErrCreateGeminiClientFailed = "创建 Gemini 客户端失败"
	ErrUnsupportedProvider      = "不支持的模型提供商: %s（后端: %s），支持: openai, claude, gemini, ollama"

	// 配置重载
	ErrConfigReloadFailed   = "配置重载失败"
	ErrRebuildExecutorFailed = "重建执行器失败"
	ErrReloadHandlerNotReady = "处理器尚未初始化，无法重载"

	// 命令执行器
	ErrWorkDirNotInProject = "工作目录必须在项目目录内: %s"
)

// New 创建一个新的错误
func New(err string) error {
	return errors.New(err)
}

// NewS 创建一个新的错误，包含原始错误信息
func NewS(errs string, err error) error {
	return fmt.Errorf("%s: %w", errs, err)
}

// NewF 创建一个格式化错误
func NewF(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
