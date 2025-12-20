package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"sync"

	"github.com/lwmacct/251207-go-pkg-cfgm/pkg/cfgm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-mcp/pkg/mcp"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool"
)

// ═══════════════════════════════════════════════════════════════════════════
// L1: Fluent Builder API
// ═══════════════════════════════════════════════════════════════════════════

// Builder Agent 链式构建器
// 提供流畅的 API 来配置和创建 Agent
//
// 内部复用 builder（functional options 的底层实现），避免重复逻辑。
//
// 使用示例：
//
//	ag, err := agent.New().
//	    Name("assistant").
//	    Model("gpt-4").
//	    APIKeyFromEnv().
//	    System("You are helpful.").
//	    Build()
type Builder struct {
	// 内部复用 functional options 的 builder
	inner *builder

	// 延迟构建
	agent *Agent     // 缓存的 Agent 实例
	built bool       // 是否已构建
	mu    sync.Mutex // 保护构建过程的互斥锁

	// 错误收集
	errs []error
}

// New 创建新的 Builder
// 这是 L1 API 的入口点
func New() *Builder {
	return &Builder{
		inner: newBuilder(),
		errs:  make([]error, 0),
	}
}

// From 从现有 Agent 创建 Builder
//
// 复制 Agent 的配置，但不共享运行时依赖。
// 新 Agent 通过配置重建 Provider 和 ToolRegistry，保证完全独立。
//
// 使用示例：
//
//	baseAgent := agent.New().Model("gpt-4").Build()
//	variantAgent := agent.From(baseAgent).
//	    Name("assistant-2").
//	    Model("gpt-4o-mini").
//	    Build()
func From(src *Agent) *Builder {
	b := New()
	// 深拷贝配置到内部 builder
	b.inner.config = cloneConfig(src.Config())
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// 身份配置
// ═══════════════════════════════════════════════════════════════════════════

// ID 设置 Agent ID
func (b *Builder) ID(id string) *Builder {
	b.inner.config.ID = id
	return b
}

// Name 设置 Agent 名称
func (b *Builder) Name(name string) *Builder {
	b.inner.config.Name = name
	return b
}

// Parent 设置父 Agent ID（用于多 Agent 协作）
func (b *Builder) Parent(parentID string) *Builder {
	b.inner.config.ParentID = parentID
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// LLM 配置
// ═══════════════════════════════════════════════════════════════════════════

// Model 设置模型名称
func (b *Builder) Model(model string) *Builder {
	b.inner.config.LLM.Model = model
	return b
}

// APIKey 设置 API 密钥
func (b *Builder) APIKey(key string) *Builder {
	b.inner.config.LLM.APIKey = key
	return b
}

// APIKeyFromEnv 从环境变量读取 API 密钥
// 如果不传参数，自动探测常见的环境变量
// 可传入自定义环境变量名，按顺序尝试
func (b *Builder) APIKeyFromEnv(envNames ...string) *Builder {
	// 默认探测的环境变量
	defaultEnvs := []string{
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"OPENROUTER_API_KEY",
		"LLM_API_KEY",
		"API_KEY",
	}

	// 合并用户指定和默认
	allEnvs := make([]string, 0, len(envNames)+len(defaultEnvs))
	allEnvs = append(allEnvs, envNames...)
	allEnvs = append(allEnvs, defaultEnvs...)

	for _, name := range allEnvs {
		if key := os.Getenv(name); key != "" {
			b.inner.config.LLM.APIKey = key
			return b
		}
	}

	b.errs = append(b.errs, errors.New("no API key found in environment variables"))
	return b
}

// BaseURL 设置 API 端点
func (b *Builder) BaseURL(url string) *Builder {
	b.inner.config.LLM.BaseURL = url
	return b
}

// MaxTokens 设置最大 token 数
func (b *Builder) MaxTokens(n int) *Builder {
	if n <= 0 {
		b.errs = append(b.errs, errors.New("maxTokens must be positive"))
		return b
	}
	b.inner.config.MaxTokens = n
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// 行为配置
// ═══════════════════════════════════════════════════════════════════════════

// System 设置系统提示词
func (b *Builder) System(prompt string) *Builder {
	b.inner.config.SystemPrompt = prompt
	return b
}

// SystemFromFile 从文件读取系统提示词
func (b *Builder) SystemFromFile(path string) *Builder {
	data, err := os.ReadFile(path) //nolint:gosec // G304: 用户提供的配置文件路径
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("read system prompt file: %w", err))
		return b
	}
	b.inner.config.SystemPrompt = string(data)
	return b
}

// WorkDir 设置工作目录
func (b *Builder) WorkDir(dir string) *Builder {
	b.inner.config.WorkDir = dir
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// 工具配置
// ═══════════════════════════════════════════════════════════════════════════

// Tools 添加工具
func (b *Builder) Tools(tools ...tool.Tool) *Builder {
	if b.inner.toolRegistry == nil {
		b.inner.toolRegistry = tool.NewRegistry()
	}
	for _, t := range tools {
		_ = b.inner.toolRegistry.Register(t)
	}
	return b
}

// ToolsFromRegistry 从注册表添加工具（按名称）
func (b *Builder) ToolsFromRegistry(registry *tool.Registry, names ...string) *Builder {
	for _, name := range names {
		if t, ok := registry.Get(name); ok {
			if b.inner.toolRegistry == nil {
				b.inner.toolRegistry = tool.NewRegistry()
			}
			_ = b.inner.toolRegistry.Register(t)
		} else {
			b.errs = append(b.errs, fmt.Errorf("tool not found: %s", name))
		}
	}
	return b
}

// ToolRegistry 设置工具注册表（替代直接传入工具）
func (b *Builder) ToolRegistry(registry *tool.Registry) *Builder {
	b.inner.toolRegistry = registry
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// MCP 服务器配置
// ═══════════════════════════════════════════════════════════════════════════

// MCPServer 添加 MCP 服务器
//
// 通过配置声明式地添加 MCP 服务器，Agent 会自动连接并加载工具。
//
// 使用示例：
//
//	ag, err := agent.New().
//	    Model("gpt-4").
//	    MCPServer(&mcp.ServerConfig{
//	        Name:    "local-tools",
//	        Command: "go",
//	        Args:    []string{"run", "cmd/mcp-server/main.go"},
//	    }).
//	    Build()
func (b *Builder) MCPServer(cfg *mcp.ServerConfig) *Builder {
	server := mcp.NewServer(cfg)
	b.inner.mcpServers = append(b.inner.mcpServers, server)
	return b
}

// MCPServers 添加多个 MCP 服务器
func (b *Builder) MCPServers(cfgs ...*mcp.ServerConfig) *Builder {
	for _, cfg := range cfgs {
		server := mcp.NewServer(cfg)
		b.inner.mcpServers = append(b.inner.mcpServers, server)
	}
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// 高级配置
// ═══════════════════════════════════════════════════════════════════════════

// Provider 直接设置 Provider（跳过自动创建）
func (b *Builder) Provider(p llm.Provider) *Builder {
	b.inner.provider = p
	return b
}

// Logger 设置日志器
func (b *Builder) Logger(logger *slog.Logger) *Builder {
	b.inner.logger = logger
	return b
}

// RetryConfig 设置重试配置
func (b *Builder) RetryConfig(cfg *RetryConfig) *Builder {
	b.inner.retryConfig = cfg
	return b
}

// MaxRetries 设置最大重试次数（便捷方法）
func (b *Builder) MaxRetries(maxRetries int) *Builder {
	if b.inner.retryConfig == nil {
		b.inner.retryConfig = DefaultRetryConfig()
	}
	b.inner.retryConfig.MaxRetries = maxRetries
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// 配置加载
// ═══════════════════════════════════════════════════════════════════════════

// FromConfig 从配置结构初始化
func (b *Builder) FromConfig(cfg *Config) *Builder {
	if cfg == nil {
		return b
	}
	// 使用 cloneConfig 确保深拷贝
	b.inner.config = cloneConfig(cfg)
	return b
}

// FromEnv 从环境变量加载配置
//
// prefix 是环境变量前缀，自动读取所有配置字段：
//   - AGENT_NAME → name
//   - AGENT_PROMPT → prompt
//   - AGENT_LLM_MODEL → llm.model
//   - AGENT_LLM_API_KEY → llm.api-key
//   - AGENT_LLM_BASE_URL → llm.base-url
//   - AGENT_MAX_TOKENS → max-tokens
//   - AGENT_WORK_DIR → work-dir
//   - ... (所有嵌套字段自动支持)
func (b *Builder) FromEnv(prefix string) *Builder {
	cfg, err := LoadConfig(
		cfgm.WithEnvPrefix(prefix+"_"),
		cfgm.WithBaseDir(""),
	)
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("load from env: %w", err))
		return b
	}
	b.applyConfig(cfg)
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// 配置文件加载 (Koanf)
// ═══════════════════════════════════════════════════════════════════════════

// FromFile 从配置文件加载配置
//
// 使用 koanf 加载 YAML 配置文件。
// 支持相对路径（基于当前工作目录）和绝对路径。
//
// 示例：
//
//	ag, err := agent.New().FromFile("config.yaml").Build()
func (b *Builder) FromFile(path string) *Builder {
	cfg, err := LoadConfig(
		cfgm.WithConfigPaths(path),
		cfgm.WithBaseDir(""), // 使用当前工作目录作为基准
	)
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("load config file: %w", err))
		return b
	}
	b.applyConfig(cfg)
	return b
}

// ToYAML 导出当前配置为 YAML 字节
//
// 使用 koanf tags 和 comment tags 生成带注释的 YAML。
func (b *Builder) ToYAML() []byte {
	return ConfigToYAML(b.inner.config)
}

// MustYAML 导出 YAML 字符串（便捷方法）
func (b *Builder) MustYAML() string {
	return string(ConfigToYAML(b.inner.config))
}

// applyConfig 应用配置到 Builder（非空字段覆盖）
func (b *Builder) applyConfig(cfg *Config) {
	if cfg.ID != "" {
		b.inner.config.ID = cfg.ID
	}
	if cfg.Name != "" {
		b.inner.config.Name = cfg.Name
	}
	if cfg.ParentID != "" {
		b.inner.config.ParentID = cfg.ParentID
	}
	// LLM 配置字段
	if cfg.LLM.Model != "" {
		b.inner.config.LLM.Model = cfg.LLM.Model
	}
	if cfg.LLM.APIKey != "" {
		b.inner.config.LLM.APIKey = cfg.LLM.APIKey
	}
	if cfg.LLM.BaseURL != "" {
		b.inner.config.LLM.BaseURL = cfg.LLM.BaseURL
	}
	if cfg.LLM.Type != "" {
		b.inner.config.LLM.Type = cfg.LLM.Type
	}
	if cfg.LLM.Timeout > 0 {
		b.inner.config.LLM.Timeout = cfg.LLM.Timeout
	}
	if cfg.LLM.MaxRetries > 0 {
		b.inner.config.LLM.MaxRetries = cfg.LLM.MaxRetries
	}
	if len(cfg.LLM.Extra) > 0 {
		if b.inner.config.LLM.Extra == nil {
			b.inner.config.LLM.Extra = make(map[string]any)
		}
		maps.Copy(b.inner.config.LLM.Extra, cfg.LLM.Extra)
	}
	// Agent 层配置
	if cfg.MaxTokens > 0 {
		b.inner.config.MaxTokens = cfg.MaxTokens
	}
	if cfg.SystemPrompt != "" {
		b.inner.config.SystemPrompt = cfg.SystemPrompt
	}
	if cfg.WorkDir != "" {
		b.inner.config.WorkDir = cfg.WorkDir
	}
	if len(cfg.Tools) > 0 {
		b.inner.config.Tools = cfg.Tools
	}
	if len(cfg.Metadata) > 0 {
		b.inner.config.Metadata = cfg.Metadata
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 便捷执行方法（自动构建）
// ═══════════════════════════════════════════════════════════════════════════

// Chat 同步对话（一次性场景）
//
// 自动构建 Agent 并执行对话，无需手动调用 Build()。
// 适用于一次性请求，Agent 会在使用后由 GC 自动回收。
//
// 使用示例：
//
//	result, err := agent.New().
//	    Model("gpt-4").
//	    APIKeyFromEnv().
//	    System("You are helpful.").
//	    Chat(ctx, "Hello")
func (b *Builder) Chat(ctx context.Context, text string) (*Result, error) {
	if err := b.ensureBuilt(); err != nil {
		return nil, err
	}
	return b.agent.Chat(ctx, text)
}

// Run 执行对话，返回事件流（支持流式/非流式）
//
// 自动构建 Agent 并执行，支持实时输出和工具调用监控。
//
// 使用示例：
//
//	// 非流式（默认）
//	for event := range agent.New().
//	    Model("gpt-4").
//	    Run(ctx, "1+1=?") {
//	    if event.Type == llm.EventTypeDone {
//	        fmt.Println(event.Result.Text)
//	    }
//	}
//
//	// 流式输出
//	for event := range agent.New().
//	    Model("gpt-4").
//	    Run(ctx, "写一首诗", WithStreaming(true)) {
//	    if event.Type == llm.EventTypeText {
//	        fmt.Print(event.Text)
//	    }
//	}
func (b *Builder) Run(ctx context.Context, text string, opts ...RunOption) <-chan *AgentEvent {
	// 创建错误通道
	errCh := make(chan *AgentEvent, 1)

	// 尝试构建
	if err := b.ensureBuilt(); err != nil {
		errCh <- &AgentEvent{
			Type:  llm.EventTypeError,
			Error: err,
		}
		close(errCh)
		return errCh
	}

	return b.agent.Run(ctx, text, opts...)
}

// Close 释放资源
//
// 仅当使用了 Chat() 或 Run() 方法时需要调用。
// 如果使用 Build() 返回 Agent，应该调用 Agent.Close()。
//
// 使用示例：
//
//	builder := agent.New().Model("gpt-4")
//	defer builder.Close()
//
//	result1, _ := builder.Chat(ctx, "Hello")
//	result2, _ := builder.Chat(ctx, "Bye")
func (b *Builder) Close() error {
	if b.agent != nil {
		return b.agent.Close()
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// 构建
// ═══════════════════════════════════════════════════════════════════════════

// Build 构建 Agent
//
// 如果没有手动设置 Provider，NewAgent 会根据配置自动创建：
//   - 优先使用 cfg.APIKey（可从 JSON 模板 {{ env "API_KEY" }} 获取）
//   - 其次从环境变量探测 API Key
func (b *Builder) Build() (*Agent, error) {
	if err := b.ensureBuilt(); err != nil {
		return nil, err
	}

	// 线程安全地返回 agent
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.agent, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// 内部构建逻辑
// ═══════════════════════════════════════════════════════════════════════════

// ensureBuilt 确保 Agent 已构建（延迟初始化，线程安全）
func (b *Builder) ensureBuilt() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 检查是否已构建（双重检查）
	if b.built {
		return nil
	}

	// 检查收集的错误
	if len(b.errs) > 0 {
		return errors.Join(b.errs...)
	}

	// 构建 Agent
	agent, err := b.buildAgent()
	if err != nil {
		return err
	}

	b.agent = agent
	b.built = true
	return nil
}

// buildAgent 内部构建方法（直接复用 newAgentFromBuilder）
func (b *Builder) buildAgent() (*Agent, error) {
	return newAgentFromBuilder(b.inner)
}
