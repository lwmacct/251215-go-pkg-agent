package agent

import (
	"testing"

	"github.com/lwmacct/251207-go-pkg-cfgm/pkg/cfgm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// DefaultConfig Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "anthropic/claude-haiku-4.5", cfg.LLM.Model)
	assert.Equal(t, "https://openrouter.ai/api/v1", cfg.LLM.BaseURL)
	assert.Equal(t, 4096, cfg.MaxTokens)
	assert.Equal(t, "You are a helpful AI assistant.", cfg.SystemPrompt)
	assert.Equal(t, ".", cfg.WorkDir)
}

// ═══════════════════════════════════════════════════════════════════════════
// ValidateConfig Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateConfig(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		cfg := &Config{
			Name:      "valid",
			MaxTokens: 1000,
		}

		err := ValidateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("zero_max_tokens_is_valid", func(t *testing.T) {
		cfg := &Config{
			MaxTokens: 0,
		}

		err := ValidateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("negative_max_tokens", func(t *testing.T) {
		cfg := &Config{
			MaxTokens: -1,
		}

		err := ValidateConfig(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max-tokens must be non-negative")
	})

	t.Run("empty_config_is_valid", func(t *testing.T) {
		cfg := &Config{}
		err := ValidateConfig(cfg)
		assert.NoError(t, err)
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// LoadConfig Tests (Koanf Integration)
// ═══════════════════════════════════════════════════════════════════════════

func TestLoadConfig(t *testing.T) {
	t.Run("defaults_only", func(t *testing.T) {
		cfg, err := LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, DefaultConfig().LLM.Model, cfg.LLM.Model)
		assert.Equal(t, DefaultConfig().MaxTokens, cfg.MaxTokens)
		assert.Equal(t, DefaultConfig().SystemPrompt, cfg.SystemPrompt)
	})

	t.Run("with_env_prefix", func(t *testing.T) {
		t.Setenv("TEST_AGENT_MAX_TOKENS", "8192")
		t.Setenv("TEST_AGENT_SYSTEM_PROMPT", "Custom prompt")

		cfg, err := LoadConfig(
			cfgm.WithEnvPrefix("TEST_AGENT_"),
		)
		require.NoError(t, err)

		assert.Equal(t, 8192, cfg.MaxTokens)
		assert.Equal(t, "Custom prompt", cfg.SystemPrompt)
	})

	t.Run("with_env_bindings", func(t *testing.T) {
		t.Setenv("CUSTOM_ENV_KEY", "custom-api-key")

		cfg, err := LoadConfig(
			cfgm.WithEnvBindings(map[string]string{
				"CUSTOM_ENV_KEY": "llm.api-key",
			}),
		)
		require.NoError(t, err)

		assert.Equal(t, "custom-api-key", cfg.LLM.APIKey)
	})

	t.Run("with_config_file", func(t *testing.T) {
		cfg, err := LoadConfig(
			cfgm.WithConfigPaths("testdata/agent.yaml"),
			cfgm.WithBaseDir(""),
		)
		require.NoError(t, err)

		assert.Equal(t, "yaml-assistant", cfg.Name)
	})
}

func TestDefaultConfigPaths(t *testing.T) {
	paths := DefaultConfigPaths()

	assert.NotEmpty(t, paths)
	assert.Contains(t, paths, ".agent.yaml")
}

// ═══════════════════════════════════════════════════════════════════════════
// ConfigToYAML Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestConfigToYAML(t *testing.T) {
	yaml := ConfigToYAML(DefaultConfig())

	assert.NotEmpty(t, yaml)
	assert.Contains(t, string(yaml), "prompt:")
	assert.Contains(t, string(yaml), "max-tokens:")
}

// ═══════════════════════════════════════════════════════════════════════════
// Template Syntax and JSON Support Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestLoadConfig_TemplateSupport(t *testing.T) {
	t.Run("yaml_with_template_syntax", func(t *testing.T) {
		// 设置测试环境变量
		t.Setenv("TEST_LLM_MODEL", "gpt-4-turbo")
		t.Setenv("TEST_API_KEY", "test-key-12345")

		cfg, err := LoadConfig(
			cfgm.WithConfigPaths("testdata/agent-template.yaml"),
			cfgm.WithBaseDir(""),
		)
		require.NoError(t, err)

		assert.Equal(t, "template-assistant", cfg.Name)
		assert.Equal(t, "gpt-4-turbo", cfg.LLM.Model)
		assert.Equal(t, "test-key-12345", cfg.LLM.APIKey)
	})

	t.Run("yaml_template_with_default_value", func(t *testing.T) {
		// 只设置 TEST_API_KEY，TEST_LLM_MODEL 使用 default 值
		t.Setenv("TEST_API_KEY", "fallback-key")

		cfg, err := LoadConfig(
			cfgm.WithConfigPaths("testdata/agent-template.yaml"),
			cfgm.WithBaseDir(""),
		)
		require.NoError(t, err)

		// 应该使用 default 值
		assert.Equal(t, "anthropic/claude-haiku-4.5", cfg.LLM.Model)
		assert.Equal(t, "fallback-key", cfg.LLM.APIKey)
	})
}

func TestLoadConfig_JSONSupport(t *testing.T) {
	t.Run("json_config_file", func(t *testing.T) {
		cfg, err := LoadConfig(
			cfgm.WithConfigPaths("testdata/agent.json"),
			cfgm.WithBaseDir(""),
		)
		require.NoError(t, err)

		assert.Equal(t, "json-assistant", cfg.Name)
		assert.Equal(t, "你是一个使用 JSON 配置的助手", cfg.SystemPrompt)
		assert.Equal(t, "anthropic/claude-haiku-4.5", cfg.LLM.Model)
		assert.Equal(t, 8192, cfg.MaxTokens)
	})
}
