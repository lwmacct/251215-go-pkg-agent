package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider"
	"github.com/lwmacct/251215-go-pkg-mcp/pkg/mcp"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool"
)

// ═══════════════════════════════════════════════════════════════════════════
// 错误定义
// ═══════════════════════════════════════════════════════════════════════════

var (
	// ErrAgentStopped Agent 已停止错误
	ErrAgentStopped = errors.New("agent is stopped")
)

// ═══════════════════════════════════════════════════════════════════════════
// Agent 基础实现
// ═══════════════════════════════════════════════════════════════════════════

// Agent Agent 基础实现
// 可以直接使用，也可以嵌入到自定义 Agent 中
//
// 核心方法:
//   - Run(): 执行对话，返回事件流，支持流式/非流式两种模式
//   - Chat(): 同步对话，等待完成（内部使用非流式）
type Agent struct {
	// 基础信息
	id       string
	name     string
	parentID string
	config   *Config

	// 核心组件
	provider     llm.Provider
	toolRegistry *tool.Registry

	// MCP 服务器
	mcpServers []*mcp.Server

	// 重试配置
	retryConfig *RetryConfig

	// 状态管理
	mu           sync.RWMutex
	state        State
	messages     []llm.Message
	stepCount    int
	lastActivity time.Time
	createdAt    time.Time

	// 生命周期
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}

	// 日志
	logger *slog.Logger
}

// NewAgent 创建新的 Agent
//
// 如果没有通过 WithProvider 设置 Provider，会根据配置自动创建：
//   - 优先使用 opts 中的 APIKey、Model、BaseURL
//   - 其次从环境变量探测 API Key
//
// 示例：
//
//	// 自动创建 Provider
//	ag, err := agent.NewAgent(
//	    agent.WithAPIKey(os.Getenv("API_KEY")),
//	    agent.WithModel("gpt-4"),
//	)
//
//	// 手动传入 Provider
//	ag, err := agent.NewAgent(
//	    agent.WithProvider(myProvider),
//	    agent.WithName("assistant"),
//	)
func NewAgent(opts ...Option) (*Agent, error) {
	// 应用选项
	b := newBuilder()
	for _, opt := range opts {
		opt(b)
	}

	return newAgentFromBuilder(b)
}

// newAgentFromBuilder 从 builder 构建 Agent（内部共享逻辑）
func newAgentFromBuilder(builder *builder) (*Agent, error) {
	// 自动创建 Provider（如果未传入）
	if builder.provider == nil {
		// 直接使用嵌套的 LLM 配置
		p, err := provider.New(&builder.config.LLM)
		if err != nil {
			return nil, fmt.Errorf("auto-create provider: %w", err)
		}
		builder.provider = p
	}

	// 验证工具名称（Fail-Fast）
	if len(builder.config.Tools) > 0 && builder.toolRegistry != nil {
		var missing []string
		for _, name := range builder.config.Tools {
			if !builder.toolRegistry.Has(name) {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("tools not found in registry: %v", missing)
		}
	}

	// 生成 ID
	id := builder.config.ID
	if id == "" {
		id = generateAgentID()
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Ensure cancel is called on error paths
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	logger := builder.logger
	if logger == nil {
		logger = slog.Default()
	}

	// 连接 MCP 服务器并加载工具
	if len(builder.mcpServers) > 0 {
		if builder.toolRegistry == nil {
			builder.toolRegistry = tool.NewRegistry()
		}
		for _, server := range builder.mcpServers {
			// 连接服务器
			if err := server.Connect(ctx); err != nil {
				// 清理已连接的服务器
				for _, s := range builder.mcpServers {
					_ = s.Close()
				}
				return nil, fmt.Errorf("connect MCP server %s: %w", server.Name(), err)
			}

			// 加载工具
			tools, err := server.LoadTools(ctx)
			if err != nil {
				// 清理已连接的服务器
				for _, s := range builder.mcpServers {
					_ = s.Close()
				}
				return nil, fmt.Errorf("load tools from MCP server %s: %w", server.Name(), err)
			}

			// 注册到工具注册表
			for _, t := range tools {
				if err := builder.toolRegistry.Register(t); err != nil {
					logger.Warn("register MCP tool failed", "server", server.Name(), "tool", t.Name(), "error", err)
				} else {
					logger.Info("registered MCP tool", "server", server.Name(), "tool", t.Name())
				}
			}
		}
	}

	agent := &Agent{
		id:           id,
		name:         builder.config.Name,
		parentID:     builder.config.ParentID,
		config:       builder.config,
		provider:     builder.provider,
		toolRegistry: builder.toolRegistry,
		mcpServers:   builder.mcpServers,
		retryConfig:  builder.retryConfig,
		state:        StateReady,
		messages:     make([]llm.Message, 0),
		createdAt:    time.Now(),
		ctx:          ctx,
		cancel:       cancel,
		stopCh:       make(chan struct{}),
		logger:       logger,
	}

	// 使用默认重试配置（如果未设置）
	if agent.retryConfig == nil {
		agent.retryConfig = DefaultRetryConfig()
	}

	// Prevent defer from calling cancel since agent owns it now
	cancel = nil

	agent.logger.Info("agent created", "id", id, "name", agent.name)
	return agent, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// 身份信息
// ═══════════════════════════════════════════════════════════════════════════

// ID 返回 Agent ID
func (a *Agent) ID() string {
	return a.id
}

// Name 返回 Agent 名称
func (a *Agent) Name() string {
	return a.name
}

// ParentID 返回父 Agent ID
func (a *Agent) ParentID() string {
	return a.parentID
}

// ═══════════════════════════════════════════════════════════════════════════
// 核心执行方法
// ═══════════════════════════════════════════════════════════════════════════

// Run 执行对话，返回事件流
//
// 支持两种执行模式：
//   - 非流式（默认）：一次性返回完整结果，适合简单问答
//   - 流式：实时返回文本增量，适合长文本生成
//
// 使用示例:
//
//	// 非流式（默认）
//	for event := range agent.Run(ctx, "1+1=?") {
//	    if event.Type == llm.EventTypeDone {
//	        fmt.Println(event.Result.Text)
//	    }
//	}
//
//	// 流式
//	for event := range agent.Run(ctx, "写一篇文章", WithStreaming(true)) {
//	    if event.Type == llm.EventTypeText {
//	        fmt.Print(event.Text)
//	    }
//	}
func (a *Agent) Run(ctx context.Context, text string, opts ...RunOption) <-chan *AgentEvent {
	eventCh := make(chan *AgentEvent, 16)

	// 应用选项
	options := ApplyRunOptions(opts...)

	go func() {
		defer close(eventCh)

		// 最外层 panic recovery
		defer func() {
			if r := recover(); r != nil {
				a.logger.Error("panic in Run goroutine",
					"panic", r,
					"agent_id", a.id,
				)
				eventCh <- &AgentEvent{
					Type:  llm.EventTypeError,
					Error: fmt.Errorf("agent panic: %v", r),
				}
			}
		}()

		// 检查状态
		a.mu.Lock()
		if a.state == StateStopped || a.state == StateStopping {
			a.mu.Unlock()
			eventCh <- &AgentEvent{Type: llm.EventTypeError, Error: ErrAgentStopped}
			return
		}
		a.state = StateRunning
		a.mu.Unlock()

		defer func() {
			a.mu.Lock()
			a.state = StateReady
			a.mu.Unlock()
		}()

		// 添加用户消息
		userMsg := llm.Message{
			Role:          llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{&llm.TextBlock{Text: text}},
		}
		a.appendMessage(userMsg)

		// 记录本轮开始位置
		startMsgIndex := len(a.messages) - 1

		// 根据模式选择执行方法
		var result *Result
		if options.Streaming {
			result = a.runLoopStreaming(ctx, eventCh, startMsgIndex)
		} else {
			result = a.runLoopBlocking(ctx, eventCh, startMsgIndex)
		}

		if result != nil {
			eventCh <- &AgentEvent{Type: llm.EventTypeDone, Result: result}
		}
	}()

	return eventCh
}

// Chat 同步对话（阻塞直到完成）
//
// 这是便捷方法，内部使用非流式模式，更高效。
// 适用于简单问答场景，不需要实时输出。
//
// 使用示例:
//
//	result, err := agent.Chat(ctx, "1+1=?")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Text)
func (a *Agent) Chat(ctx context.Context, text string) (*Result, error) {
	var result *Result
	var lastError error

	// 使用非流式模式（默认）
	for event := range a.Run(ctx, text) {
		switch event.Type {
		case llm.EventTypeDone:
			result = event.Result
		case llm.EventTypeError:
			lastError = event.Error
		}
	}

	if lastError != nil {
		return nil, lastError
	}
	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// 状态查询
// ═══════════════════════════════════════════════════════════════════════════

// Status 获取状态
func (a *Agent) Status() *Status {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &Status{
		AgentID:      a.id,
		State:        a.state,
		StepCount:    a.stepCount,
		MessageCount: len(a.messages),
		LastActivity: a.lastActivity,
	}
}

// Messages 获取消息历史
func (a *Agent) Messages() []llm.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 返回副本
	msgs := make([]llm.Message, len(a.messages))
	copy(msgs, a.messages)
	return msgs
}

// Config 返回配置的副本
//
// 返回 Agent 当前配置的深拷贝，用于以下场景：
//   - 克隆 Agent：基于现有 Agent 创建新实例
//   - 配置导出：序列化保存配置
//   - 调试检查：查看 Agent 配置状态
//
// 注意：返回的是配置快照，修改不会影响原 Agent。
func (a *Agent) Config() *Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneConfig(a.config)
}

// ═══════════════════════════════════════════════════════════════════════════
// 生命周期
// ═══════════════════════════════════════════════════════════════════════════

// Close 关闭 Agent
func (a *Agent) Close() error {
	a.mu.Lock()
	if a.state == StateStopped {
		a.mu.Unlock()
		return nil
	}
	a.state = StateStopping
	a.mu.Unlock()

	// 收集所有错误
	var errs []error

	// 发送停止信号
	close(a.stopCh)

	// 取消上下文
	a.cancel()

	// 关闭 Provider
	if a.provider != nil {
		if err := a.provider.Close(); err != nil {
			a.logger.Warn("failed to close provider", "error", err)
			errs = append(errs, fmt.Errorf("close provider: %w", err))
		}
	}

	// 关闭 MCP 服务器
	for _, server := range a.mcpServers {
		if err := server.Close(); err != nil {
			a.logger.Warn("failed to close MCP server", "server", server.Name(), "error", err)
			errs = append(errs, fmt.Errorf("close MCP server %s: %w", server.Name(), err))
		}
	}

	a.mu.Lock()
	a.state = StateStopped
	a.mu.Unlock()

	a.logger.Info("agent closed", "id", a.id)

	// 返回聚合错误
	return errors.Join(errs...)
}

// ═══════════════════════════════════════════════════════════════════════════
// 工具热加载
// ═══════════════════════════════════════════════════════════════════════════

// ErrNoToolRegistry 工具注册表未初始化错误
var ErrNoToolRegistry = errors.New("tool registry not initialized")

// ToolRegistry 返回工具注册表
//
// 返回的注册表可以直接操作工具集，支持：
//   - Register(tool): 注册或替换工具
//   - Unregister(name): 移除工具
//   - Get(name): 获取工具实例
//   - List(): 列出所有工具
//   - Has(name): 检查工具是否存在
//
// 注意：直接操作注册表的变更对后续对话立即生效。
//
// 使用示例：
//
//	registry := agent.ToolRegistry()
//	if registry.Has("calculator") {
//	    tool, _ := registry.Get("calculator")
//	    fmt.Println("Found:", tool.Name())
//	}
func (a *Agent) ToolRegistry() *tool.Registry {
	return a.toolRegistry
}

// AddTool 运行时添加或替换工具
//
// 这是热加载工具的推荐方法，适用于以下场景：
//   - 根据用户权限动态启用工具
//   - 运行时发现新的工具插件
//   - 根据任务上下文调整可用工具
//
// 如果工具已存在则替换，不存在则添加。
// 添加的工具对后续对话立即生效，但不影响正在执行的对话。
//
// 使用示例：
//
//	// 场景1: 用户升级后启用高级工具
//	if user.IsPremium() {
//	    err := agent.AddTool(&AdvancedAnalysisTool{})
//	}
//
//	// 场景2: 根据任务动态加载工具
//	if task.RequiresDatabase() {
//	    err := agent.AddTool(&DatabaseTool{connStr: task.DBConn})
//	}
func (a *Agent) AddTool(t tool.Tool) error {
	if a.toolRegistry == nil {
		return ErrNoToolRegistry
	}
	return a.toolRegistry.Register(t)
}

// RemoveTool 运行时移除工具
//
// 适用于以下场景：
//   - 用户权限降级，禁用某些工具
//   - 安全策略要求临时禁用危险工具
//   - 任务完成后清理不再需要的工具
//
// 移除后的工具在后续对话中不可用。
// 如果工具正在执行中，当前执行会完成，但新的调用会失败。
//
// 使用示例：
//
//	// 场景1: 用户权限变更
//	if user.IsDowngraded() {
//	    err := agent.RemoveTool("premium_analysis")
//	}
//
//	// 场景2: 安全策略
//	if securityAlert {
//	    err := agent.RemoveTool("file_system")
//	}
func (a *Agent) RemoveTool(name string) error {
	if a.toolRegistry == nil {
		return ErrNoToolRegistry
	}
	return a.toolRegistry.Unregister(name)
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口断言
// ═══════════════════════════════════════════════════════════════════════════

// 确保 Agent 实现了 AgentInterface 接口
var _ AgentInterface = (*Agent)(nil)
