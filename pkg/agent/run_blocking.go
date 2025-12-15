package agent

import (
	"context"
	"fmt"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 非流式执行模式
// ═══════════════════════════════════════════════════════════════════════════

// runLoopBlocking 非流式对话循环（默认）
func (a *Agent) runLoopBlocking(ctx context.Context, eventCh chan<- *AgentEvent, startMsgIndex int) *Result {
	// 循环级 panic recovery
	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("panic in runLoopBlocking",
				"panic", r,
				"agent_id", a.id,
			)
			eventCh <- &AgentEvent{
				Type:  llm.EventTypeError,
				Error: fmt.Errorf("execution loop panic: %v", r),
			}
		}
	}()

	var toolsUsed []string
	stepCount := 0

	for {
		select {
		case <-ctx.Done():
			eventCh <- &AgentEvent{Type: llm.EventTypeError, Error: ctx.Err()}
			return nil
		case <-a.stopCh:
			eventCh <- &AgentEvent{Type: llm.EventTypeError, Error: ErrAgentStopped}
			return nil
		default:
		}

		stepCount++

		// 调用 Provider（非流式）
		response, err := a.callProviderBlocking(ctx)
		if err != nil {
			eventCh <- &AgentEvent{Type: llm.EventTypeError, Error: err}
			return nil
		}

		// 添加响应消息
		a.appendMessage(response.Message)

		// 提取工具调用
		toolCalls := response.Message.GetToolCalls()
		if len(toolCalls) == 0 {
			// 无工具调用，发送完整文本事件
			text := response.Message.GetContent()
			if text != "" {
				eventCh <- &AgentEvent{Type: llm.EventTypeText, Text: text}
			}
			return a.buildResult(startMsgIndex, text, toolsUsed, stepCount)
		}

		// 发送工具调用事件
		for _, tc := range toolCalls {
			eventCh <- &AgentEvent{
				Type:     llm.EventTypeToolCall,
				ToolCall: tc,
			}
		}

		// 执行工具
		results, usedNames := a.executeToolsWithEvents(ctx, toolCalls, eventCh)
		toolsUsed = append(toolsUsed, usedNames...)

		// 添加工具结果消息
		a.appendMessage(llm.Message{
			Role:          llm.RoleUser,
			ContentBlocks: results,
		})
	}
}

// buildResult 构建对话结果
func (a *Agent) buildResult(startMsgIndex int, text string, toolsUsed []string, stepCount int) *Result {
	a.mu.RLock()
	msgs := a.messages[startMsgIndex:]
	msgsCopy := make([]llm.Message, len(msgs))
	copy(msgsCopy, msgs)
	a.mu.RUnlock()

	return &Result{
		Text:      text,
		Messages:  msgsCopy,
		ToolsUsed: toolsUsed,
		StepCount: stepCount,
	}
}

// callProviderBlocking 非流式调用 Provider
func (a *Agent) callProviderBlocking(ctx context.Context) (*llm.Response, error) {
	a.mu.RLock()
	messages := make([]llm.Message, len(a.messages))
	copy(messages, a.messages)
	a.mu.RUnlock()

	opts := a.buildProviderOptions()

	// 使用非流式 API
	return a.provider.Complete(ctx, messages, opts)
}
