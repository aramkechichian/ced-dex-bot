package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoSucceedsFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), DefaultRetryConfig(), DefaultRetryable, func(context.Context) error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestDoRetriesThenSucceeds(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}

	err := Do(context.Background(), cfg, DefaultRetryable, func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestDoStopsOnNonRetryable(t *testing.T) {
	calls := 0
	retryable := func(err error) bool { return false }

	err := Do(context.Background(), DefaultRetryConfig(), retryable, func(context.Context) error {
		calls++
		return errors.New("fatal")
	})
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
