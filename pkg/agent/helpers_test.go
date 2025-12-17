package agent

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/stretchr/testify/assert"
)

// ═══════════════════════════════════════════════════════════════════════════
// Helper Function Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short_string_unchanged",
			input:  "Hello",
			maxLen: 10,
			want:   "Hello",
		},
		{
			name:   "exact_length_unchanged",
			input:  "Hello",
			maxLen: 5,
			want:   "Hello",
		},
		{
			name:   "long_string_truncated",
			input:  "Hello World",
			maxLen: 5,
			want:   "Hello...",
		},
		{
			name:   "empty_string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "zero_max_length",
			input:  "Hello",
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateAgentID(t *testing.T) {
	t.Run("generates_unique_ids", func(t *testing.T) {
		id1 := generateAgentID()
		id2 := generateAgentID()

		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2, "IDs should be unique")
	})

	t.Run("has_correct_prefix", func(t *testing.T) {
		id := generateAgentID()
		assert.True(t, len(id) > 4, "ID should be longer than prefix")
		assert.Equal(t, "agt-", id[:4], "ID should start with 'agt-'")
	})
}

func TestCloneConfig(t *testing.T) {
	t.Run("nil_config_returns_default", func(t *testing.T) {
		result := cloneConfig(nil)

		assert.NotNil(t, result)
		assert.Equal(t, DefaultConfig().LLM.Model, result.LLM.Model)
	})

	t.Run("deep_copy_basic_fields", func(t *testing.T) {
		src := &Config{
			ID:           "src-id",
			Name:         "Source",
			ParentID:     "parent-123",
			SystemPrompt: "System prompt",
			LLM: llm.Config{
				Model:   "gpt-4",
				APIKey:  "sk-test",
				BaseURL: "https://api.test.com",
			},
			MaxTokens: 2000,
			WorkDir:   "/work",
		}

		dst := cloneConfig(src)

		assert.Equal(t, src.ID, dst.ID)
		assert.Equal(t, src.Name, dst.Name)
		assert.Equal(t, src.ParentID, dst.ParentID)
		assert.Equal(t, src.SystemPrompt, dst.SystemPrompt)
		assert.Equal(t, src.LLM.Model, dst.LLM.Model)
		assert.Equal(t, src.LLM.APIKey, dst.LLM.APIKey)
		assert.Equal(t, src.LLM.BaseURL, dst.LLM.BaseURL)
		assert.Equal(t, src.MaxTokens, dst.MaxTokens)
		assert.Equal(t, src.WorkDir, dst.WorkDir)
	})

	t.Run("deep_copy_slice", func(t *testing.T) {
		src := &Config{
			Tools: []string{"tool1", "tool2", "tool3"},
		}

		dst := cloneConfig(src)

		// Verify values are equal
		assert.ElementsMatch(t, src.Tools, dst.Tools)

		// Verify slices are independent
		dst.Tools[0] = "modified"
		assert.NotEqual(t, src.Tools[0], dst.Tools[0], "Modifying dst should not affect src")
	})

	t.Run("deep_copy_map", func(t *testing.T) {
		src := &Config{
			Metadata: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
		}

		dst := cloneConfig(src)

		// Verify values are equal
		assert.Equal(t, src.Metadata["key1"], dst.Metadata["key1"])
		assert.Equal(t, src.Metadata["key2"], dst.Metadata["key2"])

		// Verify maps are independent
		dst.Metadata["key1"] = "modified"
		assert.NotEqual(t, src.Metadata["key1"], dst.Metadata["key1"], "Modifying dst should not affect src")
	})

	t.Run("empty_slice_and_map", func(t *testing.T) {
		src := &Config{
			Name:     "Test",
			Tools:    []string{},
			Metadata: map[string]any{},
		}

		dst := cloneConfig(src)

		assert.NotNil(t, dst.Tools)
		assert.Empty(t, dst.Tools)
		assert.NotNil(t, dst.Metadata)
		assert.Empty(t, dst.Metadata)
	})

	t.Run("nil_slice_and_map", func(t *testing.T) {
		src := &Config{
			Name:     "Test",
			Tools:    nil,
			Metadata: nil,
		}

		dst := cloneConfig(src)

		// cloneConfig creates empty slices/maps for nil
		assert.NotNil(t, dst.Tools)
		assert.NotNil(t, dst.Metadata)
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Benchmark Tests
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkGenerateAgentID(b *testing.B) {
	for b.Loop() {
		_ = generateAgentID()
	}
}

func BenchmarkCloneConfig(b *testing.B) {
	src := &Config{
		ID:   "src-id",
		Name: "Source",
		LLM: llm.Config{
			Model: "gpt-4",
		},
		Tools:     []string{"tool1", "tool2", "tool3"},
		Metadata:  map[string]any{"key": "value"},
		MaxTokens: 2000,
	}

	b.ResetTimer()
	for b.Loop() {
		_ = cloneConfig(src)
	}
}

func BenchmarkTruncateString(b *testing.B) {
	longString := "This is a very long string that needs to be truncated to fit within the specified maximum length"

	b.ResetTimer()
	for b.Loop() {
		_ = truncateString(longString, 20)
	}
}
