package agent

import (
	"errors"

	"github.com/lwmacct/251207-go-pkg-mcfg/pkg/mcfg"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// Config Agent configuration
type Config struct {
	// Basic Info
	ID       string `koanf:"id"`
	Name     string `koanf:"name" comment:"Agent 名称"`
	ParentID string `koanf:"parent-id"`

	// System Prompt
	Prompt string `koanf:"prompt" comment:"系统提示词"`

	// LLM Configuration (嵌套结构，统一管理 LLM 相关配置)
	LLM llm.Config `koanf:"llm" comment:"LLM 配置"`

	// MaxTokens 最大 token 数（llm.Config 中无此字段，保留在 agent 层）
	MaxTokens int `koanf:"max-tokens" comment:"最大 token 数"`

	// Tool Configuration
	Tools []string `koanf:"tools" comment:"工具列表"`

	// Sandbox Configuration
	WorkDir string `koanf:"work-dir" comment:"工作目录"`

	// Extension Configuration
	Metadata map[string]any `koanf:"metadata"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		LLM: llm.Config{
			Model:   "anthropic/claude-haiku-4.5",
			BaseURL: "https://openrouter.ai/api/v1",
		},
		MaxTokens: 4096,
		Prompt:    "You are a helpful AI assistant.",
		WorkDir:   ".",
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Config Loading (Koanf)
// ═══════════════════════════════════════════════════════════════════════════

// LoadConfig 使用 koanf 加载配置
//
// 优先级: 默认值 < 配置文件 < 环境变量 < CLI flags
//
// 示例:
//
//	// 基础用法 (仅默认值)
//	cfg, err := agent.LoadConfig()
//
//	// 从配置文件加载
//	cfg, err := agent.LoadConfig(
//	    config.WithConfigPaths("agent.yaml"),
//	)
//
//	// 环境变量绑定
//	cfg, err := agent.LoadConfig(
//	    config.WithEnvPrefix("AGENT_"),
//	)
//
//	// 完整配置
//	cfg, err := agent.LoadConfig(
//	    config.WithConfigPaths(agent.DefaultConfigPaths()...),
//	    config.WithEnvPrefix("AGENT_"),
//	    config.WithEnvBindings(map[string]string{
//	        "OPENROUTER_API_KEY": "llm.api-key",
//	    }),
//	)
func LoadConfig(opts ...mcfg.Option) (*Config, error) {
	return mcfg.Load(*DefaultConfig(), opts...)
}

// DefaultConfigPaths 返回默认配置文件搜索路径
//
// 搜索优先级 (从高到低)：
//  1. .agent.yaml - 当前目录应用配置 (项目级别)
//  2. ~/.agent.yaml - 用户主目录配置
//  3. /etc/agent/config.yaml - 系统级别配置
//  4. config.yaml - 当前目录通用配置
//  5. config/config.yaml - 子目录配置
func DefaultConfigPaths() []string {
	return mcfg.DefaultPaths("agent")
}

// ═══════════════════════════════════════════════════════════════════════════
// Config Export
// ═══════════════════════════════════════════════════════════════════════════

// ConfigToYAML exports configuration to YAML bytes
//
// Uses koanf tags and comment tags to generate formatted YAML with comments.
func ConfigToYAML(cfg *Config) []byte {
	return mcfg.GenerateExampleYAML(*cfg)
}

// ═══════════════════════════════════════════════════════════════════════════
// Validation
// ═══════════════════════════════════════════════════════════════════════════

// ValidateConfig validates configuration
func ValidateConfig(cfg *Config) error {
	var errs []error

	if cfg.MaxTokens < 0 {
		errs = append(errs, errors.New("max-tokens must be non-negative"))
	}

	return errors.Join(errs...)
}
