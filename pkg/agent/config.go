package agent

import (
	"errors"

	"github.com/lwmacct/251207-go-pkg-mcfg/pkg/mcfg"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/urfave/cli/v3"
)

// Config Agent configuration
type Config struct {
	// Basic Info
	ID       string `koanf:"id"`
	Name     string `koanf:"name" desc:"Agent 名称"`
	ParentID string `koanf:"parent-id"`

	// System Prompt
	Prompt string `koanf:"prompt" desc:"系统提示词"`

	// LLM Configuration (嵌套结构，统一管理 LLM 相关配置)
	LLM llm.Config `koanf:"llm" desc:"LLM 配置"`

	// MaxTokens 最大 token 数（llm.Config 中无此字段，保留在 agent 层）
	MaxTokens int `koanf:"max-tokens" desc:"最大 token 数"`

	// Tool Configuration
	Tools []string `koanf:"tools" desc:"工具列表"`

	// Sandbox Configuration
	WorkDir string `koanf:"work-dir" desc:"工作目录"`

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

// AppName 是配置文件搜索的应用名称
const AppName = "agent"

// DefaultEnvBindKey 是配置文件中环境变量绑定节点的默认名称
const DefaultEnvBindKey = "envbind"

// LoadConfig 使用 koanf 加载配置
//
// 默认搜索路径: .agent.yaml, ~/.agent.yaml, /etc/agent/config.yaml 等
// 优先级: 默认值 < 配置文件 < 环境变量 < CLI flags
//
// 示例:
//
//	// 基础用法 (自动搜索默认路径)
//	cfg, err := agent.LoadConfig()
//
//	// 从指定配置文件加载
//	cfg, err := agent.LoadConfig(
//	    mcfg.WithConfigPaths("custom.yaml"),
//	)
//
//	// 环境变量自动绑定
//	cfg, err := agent.LoadConfig(
//	    mcfg.WithEnvPrefix("AGENT_"),
//	)
//
//	// 完整配置
//	cfg, err := agent.LoadConfig(
//	    mcfg.WithEnvPrefix("AGENT_"),
//	    mcfg.WithEnvBindings(map[string]string{
//	        "OPENROUTER_API_KEY": "llm.api-key",
//	    }),
//	)
func LoadConfig(opts ...mcfg.Option) (*Config, error) {
	return mcfg.Load(*DefaultConfig(), append([]mcfg.Option{
		mcfg.WithAppName(AppName),
	}, opts...)...)
}

// MustLoadConfig 加载配置，失败时 panic
//
// 适用于程序启动阶段，配置错误意味着无法继续运行。
//
// 示例:
//
//	cfg := agent.MustLoadConfig(
//	    mcfg.WithEnvPrefix("AGENT_"),
//	)
func MustLoadConfig(opts ...mcfg.Option) *Config {
	return mcfg.MustLoad(*DefaultConfig(), append([]mcfg.Option{
		mcfg.WithAppName(AppName),
	}, opts...)...)
}

// LoadConfigCmd 从 CLI 命令加载配置
//
// CLI flags 具有最高优先级，仅当用户明确指定时才覆盖其他配置源。
// 优先级: 默认值 < 配置文件 < 环境变量 < CLI flags
//
// 示例:
//
//	cmd := &cli.Command{
//	    Flags: []cli.Flag{
//	        &cli.StringFlag{Name: "name"},
//	        &cli.StringFlag{Name: "llm-model"},
//	    },
//	    Action: func(ctx context.Context, cmd *cli.Command) error {
//	        cfg, err := agent.LoadConfigCmd(cmd,
//	            mcfg.WithEnvPrefix("AGENT_"),
//	        )
//	        // ...
//	    },
//	}
func LoadConfigCmd(cmd *cli.Command, opts ...mcfg.Option) (*Config, error) {
	return mcfg.LoadCmd(cmd, *DefaultConfig(), AppName, opts...)
}

// MustLoadConfigCmd 是 LoadConfigCmd 的 panic 版本
func MustLoadConfigCmd(cmd *cli.Command, opts ...mcfg.Option) *Config {
	return mcfg.MustLoadCmd(cmd, *DefaultConfig(), AppName, opts...)
}

// LoadConfigWithEnvBind 支持从配置文件读取环境变量映射
//
// 配置文件示例:
//
//	envbind:
//	  OPENROUTER_API_KEY: llm.api-key
//	  OPENAI_API_KEY: llm.api-key
//
//	name: my-agent
//	llm:
//	  model: gpt-4
func LoadConfigWithEnvBind(opts ...mcfg.Option) (*Config, error) {
	return LoadConfig(append([]mcfg.Option{
		mcfg.WithEnvBindKey(DefaultEnvBindKey),
	}, opts...)...)
}

// DefaultConfigPaths 返回默认配置文件搜索路径
//
// Deprecated: 使用 LoadConfig() 会自动搜索默认路径，无需手动调用此函数。
//
// 搜索优先级 (从高到低)：
//  1. .agent.yaml - 当前目录应用配置 (项目级别)
//  2. ~/.agent.yaml - 用户主目录配置
//  3. /etc/agent/config.yaml - 系统级别配置
//  4. config.yaml - 当前目录通用配置
//  5. config/config.yaml - 子目录配置
func DefaultConfigPaths() []string {
	return mcfg.DefaultPaths(AppName)
}

// ═══════════════════════════════════════════════════════════════════════════
// Config Export
// ═══════════════════════════════════════════════════════════════════════════

// ConfigToYAML 导出配置为带注释的 YAML (示例格式)
//
// 使用 koanf tags 和 desc tags 生成格式化的 YAML，适合作为配置模板。
func ConfigToYAML(cfg *Config) []byte {
	return mcfg.ExampleYAML(*cfg)
}

// MarshalConfigYAML 导出配置为 YAML (无注释)
//
// 使用 koanf 原生 Marshal，输出简洁。
func MarshalConfigYAML(cfg *Config) []byte {
	return mcfg.MarshalYAML(*cfg)
}

// MarshalConfigJSON 导出配置为 JSON
func MarshalConfigJSON(cfg *Config) []byte {
	return mcfg.MarshalJSON(*cfg)
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
