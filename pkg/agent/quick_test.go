package agent

import (
	"context"
	"os"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Quick API 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestQuick(t *testing.T) {
	t.Run("Quick_should_detect_env_variables", func(t *testing.T) {
		// 跳过需要真实 API 的测试
		t.Skip("Requires real API key and provider")

		// 设置测试环境变量
		t.Setenv("OPENAI_API_KEY", "sk-test-key")
		t.Setenv("LLM_MODEL", "gpt-4o-mini")

		ctx := context.Background()

		// 零配置调用
		_, err := Quick(ctx, "Hello")
		if err != nil {
			t.Logf("Expected error in test environment: %v", err)
		}
	})

	t.Run("Quick_with_options", func(t *testing.T) {
		// 跳过需要真实 API 的测试
		t.Skip("Requires real API key and provider")

		ctx := context.Background()

		// 使用自定义选项
		_, err := Quick(ctx, "Hello",
			WithQuickModel("gpt-4"),
			WithQuickAPIKey("sk-test"),
			WithQuickSystem("You are helpful"),
			WithQuickMaxTokens(1000),
		)

		if err != nil {
			t.Logf("Expected error in test environment: %v", err)
		}
	})

	t.Run("Quick_should_fail_without_api_key", func(t *testing.T) {
		// 清除所有 API key 环境变量
		for _, key := range []string{
			"OPENAI_API_KEY",
			"ANTHROPIC_API_KEY",
			"OPENROUTER_API_KEY",
			"LLM_API_KEY",
			"API_KEY",
		} {
			_ = os.Unsetenv(key)
		}

		ctx := context.Background()

		// 应该失败（没有 API key）
		_, err := Quick(ctx, "Hello")
		if err == nil {
			t.Error("Quick() should fail without API key")
		}

		t.Logf("Expected error: %v", err)
	})
}

// TestDetectModel 测试模型探测
func TestDetectModel(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		want     string
	}{
		{
			name:     "should_detect_LLM_MODEL",
			envKey:   "LLM_MODEL",
			envValue: "gpt-4-turbo",
			want:     "gpt-4-turbo",
		},
		{
			name:     "should_detect_OPENAI_MODEL",
			envKey:   "OPENAI_MODEL",
			envValue: "gpt-4o",
			want:     "gpt-4o",
		},
		{
			name:     "should_detect_MODEL",
			envKey:   "MODEL",
			envValue: "claude-3",
			want:     "claude-3",
		},
		{
			name:     "should_use_default",
			envKey:   "",
			envValue: "",
			want:     "gpt-4o-mini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有模型环境变量
			_ = os.Unsetenv("LLM_MODEL")
			_ = os.Unsetenv("OPENAI_MODEL")
			_ = os.Unsetenv("MODEL")

			// 设置测试环境变量
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			got := detectModel()
			if got != tt.want {
				t.Errorf("detectModel() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("should_prioritize_LLM_MODEL", func(t *testing.T) {
		// 设置多个环境变量，验证优先级
		t.Setenv("LLM_MODEL", "priority-1")
		t.Setenv("OPENAI_MODEL", "priority-2")
		t.Setenv("MODEL", "priority-3")

		got := detectModel()
		if got != "priority-1" {
			t.Errorf("detectModel() should prioritize LLM_MODEL, got %v", got)
		}
	})
}

// TestDetectAPIKey 测试 API 密钥探测
func TestDetectAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		want     string
	}{
		{
			name:     "should_detect_OPENAI_API_KEY",
			envKey:   "OPENAI_API_KEY",
			envValue: "sk-openai-test",
			want:     "sk-openai-test",
		},
		{
			name:     "should_detect_ANTHROPIC_API_KEY",
			envKey:   "ANTHROPIC_API_KEY",
			envValue: "sk-anthropic-test",
			want:     "sk-anthropic-test",
		},
		{
			name:     "should_detect_OPENROUTER_API_KEY",
			envKey:   "OPENROUTER_API_KEY",
			envValue: "sk-or-test",
			want:     "sk-or-test",
		},
		{
			name:     "should_detect_LLM_API_KEY",
			envKey:   "LLM_API_KEY",
			envValue: "sk-llm-test",
			want:     "sk-llm-test",
		},
		{
			name:     "should_detect_API_KEY",
			envKey:   "API_KEY",
			envValue: "sk-generic-test",
			want:     "sk-generic-test",
		},
		{
			name:     "should_return_empty_if_not_found",
			envKey:   "",
			envValue: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有 API key 环境变量
			for _, key := range []string{
				"OPENAI_API_KEY",
				"ANTHROPIC_API_KEY",
				"OPENROUTER_API_KEY",
				"LLM_API_KEY",
				"API_KEY",
			} {
				_ = os.Unsetenv(key)
			}

			// 设置测试环境变量
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			got := detectAPIKey()
			if got != tt.want {
				t.Errorf("detectAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("should_prioritize_OPENAI_API_KEY", func(t *testing.T) {
		// 设置多个环境变量，验证优先级
		t.Setenv("OPENAI_API_KEY", "sk-priority-1")
		t.Setenv("ANTHROPIC_API_KEY", "sk-priority-2")
		t.Setenv("API_KEY", "sk-priority-3")

		got := detectAPIKey()
		if got != "sk-priority-1" {
			t.Errorf("detectAPIKey() should prioritize OPENAI_API_KEY, got %v", got)
		}
	})
}

// TestQuickOptions 测试 Quick 选项
func TestQuickOptions(t *testing.T) {
	t.Run("WithQuickModel", func(t *testing.T) {
		cfg := &quickConfig{}
		opt := WithQuickModel("gpt-4")
		opt(cfg)

		if cfg.model != "gpt-4" {
			t.Errorf("WithQuickModel() = %v, want gpt-4", cfg.model)
		}
	})

	t.Run("WithQuickAPIKey", func(t *testing.T) {
		cfg := &quickConfig{}
		opt := WithQuickAPIKey("sk-test")
		opt(cfg)

		if cfg.apiKey != "sk-test" {
			t.Errorf("WithQuickAPIKey() = %v, want sk-test", cfg.apiKey)
		}
	})

	t.Run("WithQuickSystem", func(t *testing.T) {
		cfg := &quickConfig{}
		opt := WithQuickSystem("You are helpful")
		opt(cfg)

		if cfg.system != "You are helpful" {
			t.Errorf("WithQuickSystem() = %v, want 'You are helpful'", cfg.system)
		}
	})

	t.Run("WithQuickMaxTokens", func(t *testing.T) {
		cfg := &quickConfig{}
		opt := WithQuickMaxTokens(2000)
		opt(cfg)

		if cfg.maxTokens != 2000 {
			t.Errorf("WithQuickMaxTokens() = %v, want 2000", cfg.maxTokens)
		}
	})

	t.Run("multiple_options", func(t *testing.T) {
		cfg := &quickConfig{}

		opts := []QuickOption{
			WithQuickModel("gpt-4"),
			WithQuickAPIKey("sk-test"),
			WithQuickSystem("helpful"),
			WithQuickMaxTokens(1000),
		}

		for _, opt := range opts {
			opt(cfg)
		}

		if cfg.model != "gpt-4" {
			t.Errorf("model = %v, want gpt-4", cfg.model)
		}
		if cfg.apiKey != "sk-test" {
			t.Errorf("apiKey = %v, want sk-test", cfg.apiKey)
		}
		if cfg.system != "helpful" {
			t.Errorf("system = %v, want helpful", cfg.system)
		}
		if cfg.maxTokens != 1000 {
			t.Errorf("maxTokens = %v, want 1000", cfg.maxTokens)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 示例测试（需要真实 API key，已移至 manual_test.go）
// ═══════════════════════════════════════════════════════════════════════════
