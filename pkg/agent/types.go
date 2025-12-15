package agent

import (
	"context"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// State and Result
// ═══════════════════════════════════════════════════════════════════════════

// Status Agent status snapshot
type Status struct {
	AgentID      string         `json:"agent_id"`
	State        State          `json:"state"`
	StepCount    int            `json:"step_count"`
	MessageCount int            `json:"message_count"`
	LastActivity time.Time      `json:"last_activity,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Result conversation completion result
type Result struct {
	Text        string            `json:"text"`                   // Complete response text
	Messages    []llm.Message `json:"messages,omitempty"`     // All messages in this conversation round
	ToolsUsed   []string          `json:"tools_used,omitempty"`   // List of tools used
	StepCount   int               `json:"step_count"`             // Execution steps (LLM call count)
	TotalTokens int               `json:"total_tokens,omitempty"` // Token consumption
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// Sandbox sandbox interface
type Sandbox interface {
	// WorkDir gets working directory
	WorkDir() string

	// Execute executes command in sandbox
	Execute(ctx context.Context, cmd string, args ...string) (string, error)

	// Dispose releases resources
	Dispose() error
}

// ═══════════════════════════════════════════════════════════════════════════
// Run Options
// ═══════════════════════════════════════════════════════════════════════════

// RunOptions execution options
type RunOptions struct {
	// Streaming whether to use streaming mode
	// true: returns text delta events in real-time (default)
	// false: returns complete result at once
	Streaming bool
}

// DefaultRunOptions returns default execution options
// Default uses non-streaming mode (more efficient)
func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		Streaming: false,
	}
}

// RunOption execution option function
type RunOption func(*RunOptions)

// WithStreaming sets streaming mode
//
// Example:
//
//	// Non-streaming (default) - returns at once
//	for event := range agent.Run(ctx, "1+1=?") {
//	    // Only receive tool events + 1 complete Text + Done
//	}
//
//	// Streaming - real-time output
//	for event := range agent.Run(ctx, "Write an article", WithStreaming(true)) {
//	    if event.Type == llm.EventTypeText {
//	        fmt.Print(event.Text)  // Character by character output
//	    }
//	}
func WithStreaming(enabled bool) RunOption {
	return func(o *RunOptions) {
		o.Streaming = enabled
	}
}

// ApplyRunOptions applies options
func ApplyRunOptions(opts ...RunOption) *RunOptions {
	options := DefaultRunOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// ═══════════════════════════════════════════════════════════════════════════
// Event System
// ═══════════════════════════════════════════════════════════════════════════

// AgentEvent Agent 执行事件
//
// 与 llm.Event 的区别：
//   - llm.Event: LLM 原生流式增量事件（TextDelta, ToolCallDelta）
//   - AgentEvent: Agent 聚合后的执行事件（完整 Text, ToolCall, Result）
//
// In streaming mode:
//   - Multiple llm.EventTypeText events, each containing text delta
//   - Tool call/result events
//   - Final llm.EventTypeDone event
//
// In non-streaming mode:
//   - Tool call/result events (if any)
//   - One llm.EventTypeText event with complete text
//   - Final llm.EventTypeDone event
//
// Example:
//
//	for event := range agent.Run(ctx, "Hello") {
//	    switch event.Type {
//	    case llm.EventTypeText:
//	        fmt.Print(event.Text)
//	    case llm.EventTypeToolCall:
//	        fmt.Printf("[Call: %s]\n", event.ToolCall.Name)
//	    case llm.EventTypeToolResult:
//	        fmt.Printf("[Result: %s]\n", event.ToolResult.Name)
//	    case llm.EventTypeDone:
//	        fmt.Printf("\nDone! Tools: %v\n", event.Result.ToolsUsed)
//	    case llm.EventTypeError:
//	        fmt.Printf("Error: %v\n", event.Error)
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
// Agent Interface
// ═══════════════════════════════════════════════════════════════════════════

// AgentInterface defines Agent core behavior
//
// Design notes:
//   - Run() is the only execution entry point, supports streaming/non-streaming modes
//   - Chat() is convenience method, internally uses non-streaming mode, more efficient
//   - Unified event-driven model, caller has full control over execution
type AgentInterface interface {
	// ─────────────────────────────────────────────────────────────────────
	// Identity
	// ─────────────────────────────────────────────────────────────────────

	// ID returns Agent ID
	ID() string

	// Name returns Agent name
	Name() string

	// ParentID returns parent Agent ID (for multi-Agent collaboration)
	ParentID() string

	// ─────────────────────────────────────────────────────────────────────
	// Core Execution
	// ─────────────────────────────────────────────────────────────────────

	// Run executes conversation, returns event stream
	//
	// This is Agent's core method, supports two execution modes:
	//
	// Non-streaming mode (default):
	//   - Returns complete result at once, suitable for simple Q&A
	//   - Low network overhead, low latency
	//
	// Streaming mode:
	//   - Returns text delta in real-time, suitable for long text generation
	//   - Better user experience, can see character by character output
	//
	// Example:
	//
	//	// Non-streaming (default)
	//	for event := range agent.Run(ctx, "1+1=?") {
	//	    if event.Type == llm.EventTypeDone {
	//	        fmt.Println(event.Result.Text)
	//	    }
	//	}
	//
	//	// Streaming
	//	for event := range agent.Run(ctx, "Write an article", agent.WithStreaming(true)) {
	//	    if event.Type == llm.EventTypeText {
	//	        fmt.Print(event.Text)
	//	    }
	//	}
	Run(ctx context.Context, text string, opts ...RunOption) <-chan *AgentEvent

	// Chat synchronous conversation (blocks until completion)
	//
	// This is convenience method, internally uses non-streaming mode, more efficient.
	// Suitable for simple Q&A scenarios, no need for real-time output.
	//
	// Example:
	//
	//	result, err := agent.Chat(ctx, "1+1=?")
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	fmt.Println(result.Text)
	Chat(ctx context.Context, text string) (*Result, error)

	// ─────────────────────────────────────────────────────────────────────
	// Status Query
	// ─────────────────────────────────────────────────────────────────────

	// Status gets status snapshot
	Status() *Status

	// Messages gets message history
	Messages() []llm.Message

	// ─────────────────────────────────────────────────────────────────────
	// Lifecycle
	// ─────────────────────────────────────────────────────────────────────

	// Close closes Agent, releases resources
	Close() error
}

// AgentFactory Agent factory interface
// Used by tools to create Agents (e.g., agent_create tool)
type AgentFactory interface {
	// CreateAgent creates Agent
	CreateAgent(ctx context.Context, config *Config) (AgentInterface, error)
}

// Runtime runtime interface (Agent collaboration layer, optional)
//
// Agent is independent first-class citizen, can work without Runtime.
// Runtime is optional, used for multi-Agent collaboration scenarios:
//   - Agent member management
//   - Agent lookup service
//   - Shared resource access
type Runtime interface {
	// AddAgent adds Agent to collaboration group (Agent actively joins)
	AddAgent(ag AgentInterface) error

	// RemoveAgent removes Agent from collaboration group
	RemoveAgent(agentID string)

	// CloseAgent closes and removes Agent
	CloseAgent(agentID string) error

	// GetAgent gets Agent (lookup member)
	GetAgent(agentID string) (AgentInterface, bool)

	// ListAgents lists all Agents (member list)
	ListAgents() []AgentInterface

	// ListChildAgents lists child Agents (direct subordinates)
	ListChildAgents(parentID string) []AgentInterface

	// ListDescendantAgents lists all descendant Agents (entire team)
	ListDescendantAgents(parentID string) []AgentInterface

	// GetAgentLineage gets Agent lineage (reporting chain)
	GetAgentLineage(agentID string) []string
}
