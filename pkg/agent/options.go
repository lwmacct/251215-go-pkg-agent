package agent

import (
	"log/slog"
	"os"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-mcp/pkg/mcp"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool/builtin"
)

// ═══════════════════════════════════════════════════════════════════════════
// Builder 模式
// ═══════════════════════════════════════════════════════════════════════════

// builder Agent 构建器
type builder struct {
	config       *Config
	provider     llm.Provider
	toolRegistry *tool.Registry
	logger       *slog.Logger

	// MCP 服务器
	mcpServers []*mcp.Server

	// 重试配置
	retryConfig *RetryConfig
}

// newBuilder 创建构建器
func newBuilder() *builder {
	return &builder{
		config:     DefaultConfig(),
		mcpServers: make([]*mcp.Server, 0),
	}
}

// Option Agent 配置选项
type Option func(*builder)

// ═══════════════════════════════════════════════════════════════════════════
// 基础信息选项
// ═══════════════════════════════════════════════════════════════════════════

// WithID 设置 Agent ID
func WithID(id string) Option {
	return func(b *builder) {
		b.config.ID = id
	}
}

// WithName 设置 Agent 名称
func WithName(name string) Option {
	return func(b *builder) {
		b.config.Name = name
	}
}

// WithParentID 设置父 Agent ID
func WithParentID(parentID string) Option {
	return func(b *builder) {
		b.config.ParentID = parentID
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// LLM 配置选项
// ═══════════════════════════════════════════════════════════════════════════

// WithModel 设置模型
func WithModel(model string) Option {
	return func(b *builder) {
		b.config.LLM.Model = model
	}
}

// WithAPIKey 设置 API 密钥
func WithAPIKey(apiKey string) Option {
	return func(b *builder) {
		b.config.LLM.APIKey = apiKey
	}
}

// WithAPIKeyFromEnv 从环境变量读取 API 密钥
func WithAPIKeyFromEnv(envName string) Option {
	return func(b *builder) {
		b.config.LLM.APIKey = os.Getenv(envName)
	}
}

// WithBaseURL 设置 API 端点
func WithBaseURL(baseURL string) Option {
	return func(b *builder) {
		b.config.LLM.BaseURL = baseURL
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(maxTokens int) Option {
	return func(b *builder) {
		b.config.MaxTokens = maxTokens
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Agent 行为选项
// ═══════════════════════════════════════════════════════════════════════════

// WithPrompt 设置系统提示词
func WithPrompt(prompt string) Option {
	return func(b *builder) {
		b.config.SystemPrompt = prompt
	}
}

// WithWorkDir 设置工作目录
func WithWorkDir(workDir string) Option {
	return func(b *builder) {
		b.config.WorkDir = workDir
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 依赖注入选项
// ═══════════════════════════════════════════════════════════════════════════

// WithProvider 设置 LLM Provider
//
// 如果不设置，NewAgent 会根据配置自动创建 Provider：
//   - 优先使用 WithAPIKey、WithModel、WithBaseURL 设置的值
//   - 其次从环境变量探测 API Key
//
// 示例：
//
//	// 手动传入 Provider（完全控制）
//	ag, err := agent.NewAgent(
//	    agent.WithProvider(myProvider),
//	    agent.WithName("assistant"),
//	)
//
//	// 自动创建 Provider（更简洁）
//	ag, err := agent.NewAgent(
//	    agent.WithModel("gpt-4"),
//	    agent.WithAPIKey(os.Getenv("API_KEY")),
//	)
func WithProvider(p llm.Provider) Option {
	return func(b *builder) {
		b.provider = p
	}
}

// WithToolRegistry 设置工具注册表
func WithToolRegistry(registry *tool.Registry) Option {
	return func(b *builder) {
		b.toolRegistry = registry
	}
}

// WithLogger 设置日志器
func WithLogger(logger *slog.Logger) Option {
	return func(b *builder) {
		b.logger = logger
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 便捷组合选项
// ═══════════════════════════════════════════════════════════════════════════

// WithLLM 一次性设置 LLM 相关配置
func WithLLM(model, apiKey, baseURL string) Option {
	return func(b *builder) {
		b.config.LLM.Model = model
		b.config.LLM.APIKey = apiKey
		b.config.LLM.BaseURL = baseURL
	}
}

// WithConfig 从现有配置初始化
func WithConfig(config *Config) Option {
	return func(b *builder) {
		if config == nil {
			return
		}
		// 复制配置（使用 cloneConfig 确保深拷贝）
		b.config = cloneConfig(config)
	}
}

// WithDefaults 从另一个配置继承默认值
func WithDefaults(defaults *Config) Option {
	return func(b *builder) {
		if defaults == nil {
			return
		}
		// 只继承空值
		if defaults.LLM.Model != "" && b.config.LLM.Model == "" {
			b.config.LLM.Model = defaults.LLM.Model
		}
		if defaults.LLM.APIKey != "" && b.config.LLM.APIKey == "" {
			b.config.LLM.APIKey = defaults.LLM.APIKey
		}
		if defaults.LLM.BaseURL != "" && b.config.LLM.BaseURL == "" {
			b.config.LLM.BaseURL = defaults.LLM.BaseURL
		}
		if defaults.MaxTokens > 0 && b.config.MaxTokens == 0 {
			b.config.MaxTokens = defaults.MaxTokens
		}
		if defaults.WorkDir != "" && b.config.WorkDir == "" {
			b.config.WorkDir = defaults.WorkDir
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 工具选项
// ═══════════════════════════════════════════════════════════════════════════

// WithTools 添加工具实例
//
// 在 Agent 创建时注册工具，适合以下场景：
//   - 添加自定义工具
//   - 初始化时已知需要的工具集
//   - 简单场景，不需要动态调整
//
// 注意：
//   - 创建后的工具可通过 AddTool/RemoveTool 热加载调整
//   - 如需从全局注册表引用，使用 WithGlobalTools 更灵活
//
// 使用示例：
//
//	// 场景1: 添加自定义工具
//	ag, err := agent.NewAgent(
//	    agent.WithTools(&MyCustomTool{}, &AnotherTool{}),
//	)
//
//	// 场景2: 运行时热加载（创建后动态添加）
//	ag, err := agent.NewAgent(agent.WithTools(&BaseTool{}))
//	// 后续根据需要添加更多工具
//	ag.AddTool(&AdvancedTool{})
func WithTools(tools ...tool.Tool) Option {
	return func(b *builder) {
		if b.toolRegistry == nil {
			b.toolRegistry = tool.NewRegistry()
		}
		for _, t := range tools {
			_ = b.toolRegistry.Register(t)
		}
	}
}

// WithGlobalTools 从全局注册表按名称添加工具（推荐）
//
// 通过名称引用全局注册的工具，适合标准工具的使用。
// 全局工具需要先通过 tool.Register() 注册。
//
// 使用示例：
//
//	// 使用内置工具
//	ag, err := agent.NewAgent(
//	    agent.WithGlobalTools("time", "file", "http"),
//	)
func WithGlobalTools(names ...string) Option {
	return func(b *builder) {
		if b.toolRegistry == nil {
			b.toolRegistry = builtin.Default()
		}
		b.config.Tools = names
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// MCP 服务器选项
// ═══════════════════════════════════════════════════════════════════════════

// WithMCPServer 添加 MCP 服务器
//
// 通过配置声明式地添加 MCP 服务器，Agent 会自动连接并加载工具。
//
// 使用示例：
//
//	ag, err := agent.NewAgent(
//	    agent.WithMCPServer(&mcp.ServerConfig{
//	        Name:    "local-tools",
//	        Command: "go",
//	        Args:    []string{"run", "cmd/mcp-server/main.go"},
//	    }),
//	)
func WithMCPServer(cfg *mcp.ServerConfig) Option {
	return func(b *builder) {
		server := mcp.NewServer(cfg)
		b.mcpServers = append(b.mcpServers, server)
	}
}

// WithMCPServers 添加多个 MCP 服务器
func WithMCPServers(cfgs ...*mcp.ServerConfig) Option {
	return func(b *builder) {
		for _, cfg := range cfgs {
			server := mcp.NewServer(cfg)
			b.mcpServers = append(b.mcpServers, server)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Agent 克隆选项
// ═══════════════════════════════════════════════════════════════════════════

// WithAgent 从现有 Agent 克隆配置
//
// 深拷贝源 Agent 的配置，但不共享运行时依赖。
// 新 Agent 通过配置重建 Provider 和 ToolRegistry，保证完全独立。
//
// 使用示例：
//
//	cloned, err := agent.NewAgent(
//	    agent.WithAgent(baseAgent),
//	    agent.WithName("cloned-agent"),
//	)
func WithAgent(src *Agent) Option {
	return func(b *builder) {
		// 深拷贝配置
		b.config = cloneConfig(src.Config())

		// 不共享 provider/toolRegistry
		// 通过配置重建，保证完全独立
	}
}

// CloneAgent 克隆 Agent 并应用选项
//
// 这是便捷函数，基于现有 Agent 创建新实例。
// 新 Agent 完全独立，修改配置不会影响源 Agent。
//
// 使用示例：
//
//	// 基本克隆
//	cloned, err := agent.CloneAgent(baseAgent)
//
//	// 克隆并覆盖配置
//	cloned, err := agent.CloneAgent(baseAgent,
//	    agent.WithName("cloned-agent"),
//	    agent.WithPrompt("specialized prompt"),
//	)
func CloneAgent(src *Agent, opts ...Option) (*Agent, error) {
	// 将 WithAgent 放在最前面，确保配置先被复制
	allOpts := make([]Option, 0, len(opts)+1)
	allOpts = append(allOpts, WithAgent(src))
	allOpts = append(allOpts, opts...)

	return NewAgent(allOpts...)
}

// ═══════════════════════════════════════════════════════════════════════════
// 重试配置选项
// ═══════════════════════════════════════════════════════════════════════════

// WithRetryConfig 设置工具执行重试配置
//
// 用于配置工具执行失败时的自动重试行为。
//
// 使用示例：
//
//	ag, err := agent.NewAgent(
//	    agent.WithRetryConfig(&agent.RetryConfig{
//	        MaxRetries:     3,
//	        InitialBackoff: time.Second,
//	        MaxBackoff:     10 * time.Second,
//	        Multiplier:     2.0,
//	    }),
//	)
func WithRetryConfig(cfg *RetryConfig) Option {
	return func(b *builder) {
		b.retryConfig = cfg
	}
}

// WithMaxRetries 设置最大重试次数（便捷方法）
//
// 使用默认的退避配置，只调整重试次数。
//
// 使用示例：
//
//	// 重试 3 次
//	ag, err := agent.NewAgent(agent.WithMaxRetries(3))
//
//	// 禁用重试
//	ag, err := agent.NewAgent(agent.WithMaxRetries(0))
func WithMaxRetries(max int) Option {
	return func(b *builder) {
		if b.retryConfig == nil {
			b.retryConfig = DefaultRetryConfig()
		}
		b.retryConfig.MaxRetries = max
	}
}

// DisableRetry 禁用重试（便捷方法）
//
// 等价于 WithMaxRetries(0)。
//
// 使用示例:
//
//	ag, err := agent.NewAgent(agent.DisableRetry())
func DisableRetry() Option {
	return WithMaxRetries(0)
}
