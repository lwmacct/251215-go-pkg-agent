package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 流式执行模式
// ═══════════════════════════════════════════════════════════════════════════

// runLoopStreaming 流式对话循环
func (a *Agent) runLoopStreaming(ctx context.Context, eventCh chan<- *AgentEvent, startMsgIndex int) *Result {
	// 循环级 panic recovery
	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("panic in runLoopStreaming",
				"panic", r,
				"agent_id", a.id,
			)
			eventCh <- &AgentEvent{
				Type:  llm.EventTypeError,
				Error: fmt.Errorf("streaming loop panic: %v", r),
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

		// 调用 Provider（流式）
		response, err := a.callProviderStreaming(ctx, eventCh)
		if err != nil {
			eventCh <- &AgentEvent{Type: llm.EventTypeError, Error: err}
			return nil
		}

		// 添加响应消息
		a.appendMessage(response.Message)

		// 提取工具调用
		toolCalls := response.Message.GetToolCalls()
		if len(toolCalls) == 0 {
			// 无工具调用，对话完成
			return a.buildResult(startMsgIndex, response.Message.GetContent(), toolsUsed, stepCount)
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

// callProviderStreaming 流式调用 Provider
func (a *Agent) callProviderStreaming(ctx context.Context, eventCh chan<- *AgentEvent) (*llm.Response, error) {
	a.mu.RLock()
	messages := make([]llm.Message, len(a.messages))
	copy(messages, a.messages)
	a.mu.RUnlock()

	opts := a.buildProviderOptions()

	// 使用流式 API
	chunkCh, err := a.provider.Stream(ctx, messages, opts)
	if err != nil {
		return nil, err
	}

	var textBuilder strings.Builder
	// 用于累积流式工具调用
	toolCallsMap := make(map[int]*struct {
		id   string
		name string
		args strings.Builder
	})

	for chunk := range chunkCh {
		switch chunk.Type {
		case llm.EventTypeText:
			if chunk.TextDelta != "" {
				textBuilder.WriteString(chunk.TextDelta)
				eventCh <- &AgentEvent{
					Type: llm.EventTypeText,
					Text: chunk.TextDelta,
				}
			}
		case llm.EventTypeReasoning:
			if chunk.TextDelta != "" {
				eventCh <- &AgentEvent{
					Type:      llm.EventTypeReasoning,
					Reasoning: chunk.TextDelta,
				}
			}
		case llm.EventTypeToolCall:
			if chunk.ToolCall != nil {
				tc := chunk.ToolCall
				// 获取或创建工具调用条目
				entry, exists := toolCallsMap[tc.Index]
				if !exists {
					entry = &struct {
						id   string
						name string
						args strings.Builder
					}{}
					toolCallsMap[tc.Index] = entry
				}
				// 累积数据
				if tc.ID != "" {
					entry.id = tc.ID
				}
				if tc.Name != "" {
					entry.name = tc.Name
				}
				if tc.ArgumentsDelta != "" {
					entry.args.WriteString(tc.ArgumentsDelta)
				}
			}
		case llm.EventTypeToolResult, llm.EventTypeThinking, llm.EventTypeDone, llm.EventTypeError:
			// 这些事件类型在流式块处理中不出现，由上层处理
		}
	}

	// 将累积的工具调用转换为 ContentBlocks
	toolCallBlocks := make([]*llm.ToolCall, 0, len(toolCallsMap))
	for i := range len(toolCallsMap) {
		if entry, exists := toolCallsMap[i]; exists {
			// 解析 JSON 参数
			var input map[string]any
			if argsStr := entry.args.String(); argsStr != "" {
				if err := json.Unmarshal([]byte(argsStr), &input); err != nil {
					a.logger.Warn("failed to parse tool call arguments",
						"name", entry.name,
						"error", err,
					)
					input = make(map[string]any)
				}
			}
			toolCallBlocks = append(toolCallBlocks, &llm.ToolCall{
				ID:    entry.id,
				Name:  entry.name,
				Input: input,
			})
		}
	}

	// 构建响应消息 - 将工具调用添加到 ContentBlocks
	contentBlocks := make([]llm.ContentBlock, 0, 1+len(toolCallBlocks))
	if textBuilder.Len() > 0 {
		contentBlocks = append(contentBlocks, &llm.TextBlock{Text: textBuilder.String()})
	}
	// 添加工具调用块
	for _, toolBlock := range toolCallBlocks {
		contentBlocks = append(contentBlocks, toolBlock)
	}

	msg := llm.Message{
		Role:          llm.RoleAssistant,
		ContentBlocks: contentBlocks,
	}

	return &llm.Response{Message: msg}, nil
}
