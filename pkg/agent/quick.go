package agent

import (
	"context"
	"os"
)

// ═══════════════════════════════════════════════════════════════════════════
// L0: 函数式 API - 极简一次性调用
// ═══════════════════════════════════════════════════════════════════════════

// Quick 快速对话（零配置）
//
// 自动从环境变量读取 API 密钥和模型配置，适合脚本和快速原型。
//
// 环境变量探测顺序：
//   - API Key: OPENAI_API_KEY, ANTHROPIC_API_KEY, OPENROUTER_API_KEY, LLM_API_KEY, API_KEY
//   - Model: LLM_MODEL, OPENAI_MODEL, MODEL (默认: gpt-4o-mini)
//
// 使用示例：
//
//	// 最简单的调用（使用环境变量）
//	result, err := agent.Quick(ctx, "翻译成法语: Hello")
//
//	// 自定义配置
//	result, err := agent.Quick(ctx, "写一首诗",
//	    agent.WithQuickModel("gpt-4"),
//	    agent.WithQuickSystem("你是一位诗人"),
//	)
func Quick(ctx context.Context, message string, opts ...QuickOption) (*Result, error) {
	// 默认配置
	cfg := &quickConfig{
		model:  detectModel(),
		apiKey: detectAPIKey(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(cfg)
	}

	// 使用 Builder 构建并执行
	return New().
		Model(cfg.model).
		APIKey(cfg.apiKey).
		System(cfg.system).
		MaxTokens(cfg.maxTokens).
		Chat(ctx, message)
}

// ═══════════════════════════════════════════════════════════════════════════
// 配置选项
// ═══════════════════════════════════════════════════════════════════════════

// quickConfig 快速调用的配置
type quickConfig struct {
	model     string
	apiKey    string
	system    string
	maxTokens int
}

// QuickOption 快速调用的配置选项
type QuickOption func(*quickConfig)

// WithQuickModel 设置模型
func WithQuickModel(model string) QuickOption {
	return func(c *quickConfig) {
		c.model = model
	}
}

// WithQuickAPIKey 设置 API 密钥
func WithQuickAPIKey(apiKey string) QuickOption {
	return func(c *quickConfig) {
		c.apiKey = apiKey
	}
}

// WithQuickSystem 设置系统提示词
func WithQuickSystem(prompt string) QuickOption {
	return func(c *quickConfig) {
		c.system = prompt
	}
}

// WithQuickMaxTokens 设置最大 token 数
func WithQuickMaxTokens(n int) QuickOption {
	return func(c *quickConfig) {
		c.maxTokens = n
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 环境变量探测
// ═══════════════════════════════════════════════════════════════════════════

// detectModel 探测模型配置
func detectModel() string {
	// 优先从专用环境变量读取
	if model := os.Getenv("LLM_MODEL"); model != "" {
		return model
	}
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		return model
	}
	if model := os.Getenv("MODEL"); model != "" {
		return model
	}

	// 默认模型
	return "gpt-4o-mini"
}

// detectAPIKey 探测 API 密钥
func detectAPIKey() string {
	// 按常见程度排序
	envs := []string{
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"OPENROUTER_API_KEY",
		"LLM_API_KEY",
		"API_KEY",
	}

	for _, name := range envs {
		if key := os.Getenv(name); key != "" {
			return key
		}
	}

	return ""
}
