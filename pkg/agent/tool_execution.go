package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool"
)

// ═══════════════════════════════════════════════════════════════════════════
// 工具执行
// ═══════════════════════════════════════════════════════════════════════════

// executeToolsWithEvents 执行工具并发送事件
func (a *Agent) executeToolsWithEvents(ctx context.Context, toolCalls []*llm.ToolCall, eventCh chan<- *AgentEvent) ([]llm.ContentBlock, []string) {
	if a.toolRegistry == nil {
		a.logger.Error("tool registry not configured")
		return nil, nil
	}

	results := make([]llm.ContentBlock, 0, len(toolCalls))
	usedNames := make([]string, 0, len(toolCalls))

	a.logger.Info("executing tools", "count", len(toolCalls))

	for _, tc := range toolCalls {
		usedNames = append(usedNames, tc.Name)

		a.logger.Info("tool call", "tool", tc.Name, "id", tc.ID)

		// 单个工具执行的 panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					a.logger.Error("panic in tool execution",
						"panic", r,
						"tool", tc.Name,
						"agent_id", a.id,
					)
					tr := &llm.ToolResult{
						ToolID:  tc.ID,
						Name:    tc.Name,
						Content: fmt.Sprintf("Tool execution panic: %v", r),
						IsError: true,
					}
					eventCh <- &AgentEvent{Type: llm.EventTypeToolResult, ToolResult: tr}
					results = append(results, &llm.ToolResultBlock{
						ToolUseID: tc.ID,
						Content:   tr.Content,
						IsError:   true,
					})
				}
			}()

			t, ok := a.toolRegistry.Get(tc.Name)
			if !ok {
				a.logger.Warn("tool not found", "tool", tc.Name)
				tr := &llm.ToolResult{
					ToolID:  tc.ID,
					Name:    tc.Name,
					Content: fmt.Sprintf("Error: tool '%s' not found", tc.Name),
					IsError: true,
				}
				eventCh <- &AgentEvent{Type: llm.EventTypeToolResult, ToolResult: tr}
				results = append(results, &llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   tr.Content,
					IsError:   true,
				})
				return // 闭包内使用 return 而不是 continue
			}

			// 序列化参数
			inputJSON, err := json.Marshal(tc.Input)
			if err != nil {
				a.logger.Error("failed to marshal arguments", "error", err)
				tr := &llm.ToolResult{
					ToolID:  tc.ID,
					Name:    tc.Name,
					Content: fmt.Sprintf("Error: failed to marshal arguments: %v", err),
					IsError: true,
				}
				eventCh <- &AgentEvent{Type: llm.EventTypeToolResult, ToolResult: tr}
				results = append(results, &llm.ToolResultBlock{
					ToolUseID: tc.ID,
					Content:   tr.Content,
					IsError:   true,
				})
				return // 闭包内使用 return 而不是 continue
			}

			// 将 AgentID 存入 context
			ctx := tool.ContextWithAgentID(ctx, a.id)

			// 执行工具（优先使用 ExecuteResult）
			a.logger.Debug("executing tool", "tool", tc.Name)

			var output any
			var execErr error
			var metadata tool.Metadata
			var retries int

			// 定义工具执行操作
			operation := func() (any, error) {
				// 检查是否实现了 ResultExecutor 接口
				if re, ok := t.(tool.ResultExecutor); ok {
					result := re.ExecuteResult(ctx, inputJSON)
					if result.IsErr() {
						return nil, result.Error()
					}
					metadata = result.Meta()
					return result.Value(), nil
				} else {
					// 兼容旧工具
					return t.Execute(ctx, inputJSON)
				}
			}

			// 使用重试机制执行工具
			if a.retryConfig != nil && a.retryConfig.MaxRetries > 0 {
				output, retries, execErr = a.retryWithBackoff(ctx, operation, a.retryConfig)
			} else {
				// 不重试，直接执行
				output, execErr = operation()
			}

			// 更新元数据中的重试次数
			if metadata.Retries == 0 {
				metadata.Retries = retries
			}

			var content string
			var isError bool
			if execErr != nil {
				a.logger.Error("tool execution failed", "tool", tc.Name, "error", execErr)
				content = fmt.Sprintf("Error: %v", execErr)
				isError = true
			} else {
				jsonBytes, _ := json.Marshal(output)
				content = string(jsonBytes)
			}

			// 记录元数据（如果有）
			if metadata.ToolName != "" || metadata.Duration > 0 {
				logAttrs := []any{"tool", tc.Name}
				if metadata.Duration > 0 {
					logAttrs = append(logAttrs, "duration", metadata.Duration)
				}
				if metadata.Cached {
					logAttrs = append(logAttrs, "cached", true)
				}
				if metadata.Retries > 0 {
					logAttrs = append(logAttrs, "retries", metadata.Retries)
				}
				a.logger.Debug("tool metadata", logAttrs...)
			}

			a.logger.Info("tool result", "tool", tc.Name, "result_preview", truncateString(content, 200))

			tr := &llm.ToolResult{
				ToolID:  tc.ID,
				Name:    tc.Name,
				Content: content,
				IsError: isError,
			}
			eventCh <- &AgentEvent{Type: llm.EventTypeToolResult, ToolResult: tr}
			results = append(results, &llm.ToolResultBlock{
				ToolUseID: tc.ID,
				Content:   content,
				IsError:   isError,
			})
		}() // 闭包结束
	}

	a.logger.Info("tools executed", "count", len(results))
	return results, usedNames
}
