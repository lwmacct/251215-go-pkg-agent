package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool"
)

// ═══════════════════════════════════════════════════════════════════════════
// 内部辅助方法
// ═══════════════════════════════════════════════════════════════════════════

// appendMessage 线程安全地添加消息
func (a *Agent) appendMessage(msg llm.Message) {
	a.mu.Lock()
	a.messages = append(a.messages, msg)
	a.stepCount++
	a.lastActivity = time.Now()
	a.mu.Unlock()
}

// buildProviderOptions 构建 Provider 选项
func (a *Agent) buildProviderOptions() *llm.Options {
	opts := &llm.Options{
		System:      a.config.Prompt,
		MaxTokens:   a.config.MaxTokens,
		Temperature: 0.7,
	}

	// 添加工具 Schema
	if a.toolRegistry != nil && a.toolRegistry.Count() > 0 {
		tools := make([]llm.ToolSchema, 0)
		for _, t := range a.toolRegistry.List() {
			toolSchema := llm.ToolSchema{
				Name:        t.Name(),
				Description: t.Description(),
				InputSchema: t.InputSchema(),
			}

			// 提取 Examples（如果工具实现了 Documentable）
			if doc, ok := t.(tool.Documentable); ok {
				examples := doc.Examples()
				if len(examples) > 0 {
					// 转换为 InputExamples（只提取 Input 部分）
					inputExamples := make([]any, 0, len(examples))
					for _, ex := range examples {
						inputExamples = append(inputExamples, ex.Input)
					}
					toolSchema.InputExamples = inputExamples
				}
			}

			tools = append(tools, toolSchema)
		}
		opts.Tools = tools

		// 注入工具手册
		a.injectToolManual(opts)
	}

	return opts
}

// injectToolManual 注入工具手册
func (a *Agent) injectToolManual(opts *llm.Options) {
	if strings.Contains(opts.System, "### Tools Manual") {
		return
	}

	var lines []string
	for _, t := range a.toolRegistry.List() {
		lines = append(lines, fmt.Sprintf("- `%s`: %s", t.Name(), t.Description()))
	}

	if len(lines) > 0 {
		manualSection := "\n\n### Tools Manual\n\n" +
			"The following tools are available:\n\n" +
			strings.Join(lines, "\n")
		opts.System += manualSection
	}
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateAgentID 生成 Agent ID
func generateAgentID() string {
	return "agt-" + uuid.New().String()
}

// cloneConfig 深拷贝 Config
//
// 用于 Agent 克隆，确保配置完全独立，避免互相影响。
// 注意：切片和 map 会被深拷贝，但 map 中的值是浅拷贝。
func cloneConfig(src *Config) *Config {
	if src == nil {
		return DefaultConfig()
	}

	// 深拷贝切片
	tools := make([]string, len(src.Tools))
	copy(tools, src.Tools)

	// 深拷贝 map (值浅拷贝)
	metadata := make(map[string]any, len(src.Metadata))
	for k, v := range src.Metadata {
		metadata[k] = v
	}

	// 深拷贝 LLM.Extra map
	var llmExtra map[string]any
	if src.LLM.Extra != nil {
		llmExtra = make(map[string]any, len(src.LLM.Extra))
		for k, v := range src.LLM.Extra {
			llmExtra[k] = v
		}
	}

	return &Config{
		ID:        src.ID,
		Name:      src.Name,
		ParentID:  src.ParentID,
		Prompt:    src.Prompt,
		LLM: llm.Config{
			Type:       src.LLM.Type,
			APIKey:     src.LLM.APIKey,
			Model:      src.LLM.Model,
			BaseURL:    src.LLM.BaseURL,
			Timeout:    src.LLM.Timeout,
			MaxRetries: src.LLM.MaxRetries,
			Extra:      llmExtra,
		},
		MaxTokens: src.MaxTokens,
		Tools:     tools,
		WorkDir:   src.WorkDir,
		Metadata:  metadata,
	}
}
