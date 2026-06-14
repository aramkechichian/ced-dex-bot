package resilience

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// RetryConfig controls exponential backoff retries.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig returns sensible RPC retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}
}

// Retryable decides whether a failed attempt should be retried.
type Retryable func(error) bool

// DefaultRetryable retries unless the context was cancelled.
func DefaultRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// Do runs fn with exponential backoff until success or attempts exhausted.
func Do(ctx context.Context, cfg RetryConfig, retryable Retryable, fn func(context.Context) error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if retryable == nil {
		retryable = DefaultRetryable
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 200 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 2 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		if !retryable(lastErr) || attempt == cfg.MaxAttempts-1 {
			return lastErr
		}

		delay := backoffDelay(cfg, attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("retry exhausted: %w", lastErr)
}

func backoffDelay(cfg RetryConfig, attempt int) time.Duration {
	delay := cfg.BaseDelay * time.Duration(1<<attempt)
	if delay > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return delay
}
