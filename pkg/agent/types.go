package agent

import (
	"context"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 状态与结果
// ═══════════════════════════════════════════════════════════════════════════

// Status Agent 状态快照
type Status struct {
	AgentID      string         `json:"agent_id"`
	State        State          `json:"state"`
	StepCount    int            `json:"step_count"`
	MessageCount int            `json:"message_count"`
	LastActivity time.Time      `json:"last_activity,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Result 对话完成结果
type Result struct {
	Text        string        `json:"text"`                   // 完整响应文本
	Messages    []llm.Message `json:"messages,omitempty"`     // 本轮对话的所有消息
	ToolsUsed   []string      `json:"tools_used,omitempty"`   // 使用过的工具列表
	StepCount   int           `json:"step_count"`             // 执行步数（LLM 调用次数）
	TotalTokens int           `json:"total_tokens,omitempty"` // Token 消耗
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Sandbox 沙箱接口
type Sandbox interface {
	// WorkDir 获取工作目录
	WorkDir() string

	// Execute 在沙箱中执行命令
	Execute(ctx context.Context, cmd string, args ...string) (string, error)

	// Dispose 释放资源
	Dispose() error
}

// ═══════════════════════════════════════════════════════════════════════════
// 执行选项
// ═══════════════════════════════════════════════════════════════════════════

// RunOptions 执行选项
type RunOptions struct {
	// Streaming 是否使用流式模式
	// true: 实时返回文本增量事件
	// false: 一次性返回完整结果（默认）
	Streaming bool
}

// DefaultRunOptions 返回默认执行选项
// 默认使用非流式模式（效率更高）
func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		Streaming: false,
	}
}

// RunOption 执行选项函数
type RunOption func(*RunOptions)

// WithStreaming 设置流式模式
//
// 示例：
//
//	// 非流式（默认）- 一次性返回
//	for event := range agent.Run(ctx, "1+1=?") {
//	    // 只收到工具事件 + 1 个完整 Text + Done
//	}
//
//	// 流式 - 实时输出
//	for event := range agent.Run(ctx, "写一篇文章", WithStreaming(true)) {
//	    if event.Type == llm.EventTypeText {
//	        fmt.Print(event.Text)  // 逐字输出
//	    }
//	}
func WithStreaming(enabled bool) RunOption {
	return func(o *RunOptions) {
		o.Streaming = enabled
	}
}

// ApplyRunOptions 应用选项
func ApplyRunOptions(opts ...RunOption) *RunOptions {
	options := DefaultRunOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// ═══════════════════════════════════════════════════════════════════════════
// 事件系统
// ═══════════════════════════════════════════════════════════════════════════

// AgentEvent Agent 执行事件
//
// 与 llm.Event 的区别：
//   - llm.Event: LLM 原生流式增量事件（TextDelta, ToolCallDelta）
//   - AgentEvent: Agent 聚合后的执行事件（完整 Text, ToolCall, Result）
//
// 流式模式：
//   - 多个 llm.EventTypeText 事件，每个包含文本增量
//   - 工具调用/结果事件
//   - 最终 llm.EventTypeDone 事件
//
// 非流式模式：
//   - 工具调用/结果事件（如有）
//   - 一个 llm.EventTypeText 事件包含完整文本
//   - 最终 llm.EventTypeDone 事件
//
// 示例：
//
//	for event := range agent.Run(ctx, "Hello") {
//	    switch event.Type {
//	    case llm.EventTypeText:
//	        fmt.Print(event.Text)
//	    case llm.EventTypeToolCall:
//	        fmt.Printf("[调用: %s]\n", event.ToolCall.Name)
//	    case llm.EventTypeToolResult:
//	        fmt.Printf("[结果: %s]\n", event.ToolResult.Name)
//	    case llm.EventTypeDone:
//	        fmt.Printf("\n完成! 工具: %v\n", event.Result.ToolsUsed)
//	    case llm.EventTypeError:
//	        fmt.Printf("错误: %v\n", event.Error)
//	    }
//	}
type AgentEvent struct {
	Type llm.EventType `json:"type"`

	// llm.EventTypeText
	Text string `json:"text,omitempty"`

	// llm.EventTypeToolCall
	ToolCall *llm.ToolCall `json:"tool_call,omitempty"`

	// llm.EventTypeToolResult
	ToolResult *llm.ToolResult `json:"tool_result,omitempty"`

	// llm.EventTypeReasoning
	Reasoning string `json:"reasoning,omitempty"`

	// llm.EventTypeDone
	Result *Result `json:"result,omitempty"`

	// llm.EventTypeError
	Error error `json:"error,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════
// Agent 接口
// ═══════════════════════════════════════════════════════════════════════════

// AgentInterface 定义 Agent 核心行为
//
// 设计说明：
//   - Run() 是唯一执行入口，支持流式/非流式模式
//   - Chat() 是便捷方法，内部使用非流式模式，效率更高
//   - 统一的事件驱动模型，调用者拥有完全控制权
type AgentInterface interface {
	// ─────────────────────────────────────────────────────────────────────
	// 身份信息
	// ─────────────────────────────────────────────────────────────────────

	// ID 返回 Agent ID
	ID() string

	// Name 返回 Agent 名称
	Name() string

	// ParentID 返回父 Agent ID（多 Agent 协作场景）
	ParentID() string

	// ─────────────────────────────────────────────────────────────────────
	// 核心执行
	// ─────────────────────────────────────────────────────────────────────

	// Run 执行对话，返回事件流
	//
	// 这是 Agent 的核心方法，支持两种执行模式：
	//
	// 非流式模式（默认）：
	//   - 一次性返回完整结果，适合简单问答
	//   - 网络开销低，延迟低
	//
	// 流式模式：
	//   - 实时返回文本增量，适合长文本生成
	//   - 用户体验更好，可以看到逐字输出
	//
	// 示例：
	//
	//	// 非流式（默认）
	//	for event := range agent.Run(ctx, "1+1=?") {
	//	    if event.Type == llm.EventTypeDone {
	//	        fmt.Println(event.Result.Text)
	//	    }
	//	}
	//
	//	// 流式
	//	for event := range agent.Run(ctx, "写一篇文章", agent.WithStreaming(true)) {
	//	    if event.Type == llm.EventTypeText {
	//	        fmt.Print(event.Text)
	//	    }
	//	}
	Run(ctx context.Context, text string, opts ...RunOption) <-chan *AgentEvent

	// Chat 同步对话（阻塞直到完成）
	//
	// 这是便捷方法，内部使用非流式模式，效率更高。
	// 适合简单问答场景，无需实时输出。
	//
	// 示例：
	//
	//	result, err := agent.Chat(ctx, "1+1=?")
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	fmt.Println(result.Text)
	Chat(ctx context.Context, text string) (*Result, error)

	// ─────────────────────────────────────────────────────────────────────
	// 状态查询
	// ─────────────────────────────────────────────────────────────────────

	// Status 获取状态快照
	Status() *Status

	// Messages 获取消息历史
	Messages() []llm.Message

	// ─────────────────────────────────────────────────────────────────────
	// 生命周期
	// ─────────────────────────────────────────────────────────────────────

	// Close 关闭 Agent，释放资源
	Close() error
}

// AgentFactory Agent 工厂接口
// 供工具创建 Agent 使用（如 agent_create 工具）
type AgentFactory interface {
	// CreateAgent 创建 Agent
	CreateAgent(ctx context.Context, config *Config) (AgentInterface, error)
}

// Runtime 运行时接口（Agent 协作层，可选）
//
// Agent 是独立的一等公民，可以脱离 Runtime 独立工作。
// Runtime 是可选的，用于多 Agent 协作场景：
//   - Agent 成员管理
//   - Agent 查找服务
//   - 共享资源访问
type Runtime interface {
	// AddAgent 添加 Agent 到协作组（Agent 主动加入）
	AddAgent(ag AgentInterface) error

	// RemoveAgent 从协作组移除 Agent
	RemoveAgent(agentID string)

	// CloseAgent 关闭并移除 Agent
	CloseAgent(agentID string) error

	// GetAgent 获取 Agent（查找成员）
	GetAgent(agentID string) (AgentInterface, bool)

	// ListAgents 列出所有 Agent（成员列表）
	ListAgents() []AgentInterface

	// ListChildAgents 列出子 Agent（直接下属）
	ListChildAgents(parentID string) []AgentInterface

	// ListDescendantAgents 列出所有后代 Agent（整个团队）
	ListDescendantAgents(parentID string) []AgentInterface

	// GetAgentLineage 获取 Agent 血统链（上报链路）
	GetAgentLineage(agentID string) []string
}
