package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ═══════════════════════════════════════════════════════════════════════════
// Retry Config Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 2, cfg.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 5*time.Second, cfg.MaxBackoff)
	assert.InDelta(t, 2.0, cfg.Multiplier, 0.001)
}

// ═══════════════════════════════════════════════════════════════════════════
// IsRetriable Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil_error",
			err:  nil,
			want: false,
		},
		{
			name: "timeout_error",
			err:  errors.New("connection timeout"),
			want: true,
		},
		{
			name: "connection_refused",
			err:  errors.New("dial tcp: connection refused"),
			want: true,
		},
		{
			name: "temporary_failure",
			err:  errors.New("temporary failure in name resolution"),
			want: true,
		},
		{
			name: "rate_limit",
			err:  errors.New("rate limit exceeded"),
			want: true,
		},
		{
			name: "429_status",
			err:  errors.New("HTTP 429: Too Many Requests"),
			want: true,
		},
		{
			name: "503_status",
			err:  errors.New("HTTP 503: Service Unavailable"),
			want: true,
		},
		{
			name: "context_deadline_exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "permanent_error",
			err:  errors.New("invalid API key"),
			want: false,
		},
		{
			name: "not_found_error",
			err:  errors.New("resource not found"),
			want: false,
		},
		{
			name: "validation_error",
			err:  errors.New("invalid input parameter"),
			want: false,
		},
		{
			name: "case_insensitive_timeout",
			err:  errors.New("TIMEOUT occurred"),
			want: true,
		},
		{
			name: "case_insensitive_rate_limit",
			err:  errors.New("RATE LIMIT exceeded"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetriable(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsRetriable_EdgeCases(t *testing.T) {
	t.Run("embedded_pattern", func(t *testing.T) {
		err := errors.New("upstream server returned timeout error at 10:30")
		assert.True(t, IsRetriable(err))
	})

	t.Run("multiple_patterns", func(t *testing.T) {
		err := errors.New("rate limit exceeded, please retry after timeout")
		assert.True(t, IsRetriable(err))
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// RetryConfig Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestRetryConfig(t *testing.T) {
	t.Run("custom_config", func(t *testing.T) {
		cfg := &RetryConfig{
			MaxRetries:     5,
			InitialBackoff: time.Second,
			MaxBackoff:     30 * time.Second,
			Multiplier:     1.5,
		}

		assert.Equal(t, 5, cfg.MaxRetries)
		assert.Equal(t, time.Second, cfg.InitialBackoff)
		assert.Equal(t, 30*time.Second, cfg.MaxBackoff)
		assert.InDelta(t, 1.5, cfg.Multiplier, 0.001)
	})

	t.Run("zero_retries", func(t *testing.T) {
		cfg := &RetryConfig{
			MaxRetries: 0,
		}

		assert.Equal(t, 0, cfg.MaxRetries, "Zero retries should be valid (no retry)")
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Benchmark Tests
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkIsRetriable(b *testing.B) {
	err := errors.New("connection timeout")

	for b.Loop() {
		_ = IsRetriable(err)
	}
}

func BenchmarkIsRetriable_NoMatch(b *testing.B) {
	err := errors.New("invalid API key")

	for b.Loop() {
		_ = IsRetriable(err)
	}
}
