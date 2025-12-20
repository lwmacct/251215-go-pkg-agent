package agent

import (
	"context"
	"strings"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries     int           // 最大重试次数（0 表示不重试）
	InitialBackoff time.Duration // 初始退避时间
	MaxBackoff     time.Duration // 最大退避时间
	Multiplier     float64       // 退避倍数（指数退避）
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     2, // 最多重试 2 次（总共执行 3 次）
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		Multiplier:     2.0,
	}
}

// IsRetriable 判断错误是否可重试
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// 可重试的错误模式
	retriablePatterns := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"rate limit",
		"429",
		"503",
		"context deadline exceeded",
	}

	for _, pattern := range retriablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// retryWithBackoff 使用指数退避重试执行操作
func (a *Agent) retryWithBackoff(
	ctx context.Context,
	operation func() (any, error),
	cfg *RetryConfig,
) (any, int, error) {
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result, err := operation()
		if err == nil {
			return result, attempt, nil
		}

		lastErr = err

		// 检查是否可重试
		if !IsRetriable(err) {
			a.logger.Debug("error not retriable", "error", err, "attempt", attempt)
			return nil, attempt, err
		}

		// 达到最大重试次数
		if attempt >= cfg.MaxRetries {
			a.logger.Warn("max retries reached", "max_retries", cfg.MaxRetries, "error", err)
			break
		}

		// 退避等待
		a.logger.Info("retrying after backoff", "attempt", attempt+1, "backoff", backoff, "error", err)

		select {
		case <-ctx.Done():
			return nil, attempt, ctx.Err()
		case <-time.After(backoff):
			backoff = min(time.Duration(float64(backoff)*cfg.Multiplier), cfg.MaxBackoff)
		}
	}

	return nil, cfg.MaxRetries, lastErr
}
