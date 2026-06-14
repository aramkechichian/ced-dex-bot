package resilience

import (
	"context"
	"log/slog"
	"math/big"

	"github.com/aramik/ced-dex-bot/internal/uniswap"
)

// RetryingQuoter wraps a Uniswap quoter with RPC retry + backoff.
type RetryingQuoter struct {
	inner  uniswap.Quoter
	cfg    RetryConfig
	log    *slog.Logger
}

// NewRetryingQuoter creates a quoter that retries transient RPC failures.
func NewRetryingQuoter(inner uniswap.Quoter, cfg RetryConfig, log *slog.Logger) *RetryingQuoter {
	if cfg.MaxAttempts <= 0 {
		cfg = DefaultRetryConfig()
	}
	return &RetryingQuoter{
		inner: inner,
		cfg:   cfg,
		log:   log,
	}
}

// QuoteSellETH retries QuoteSellETH on transient failures.
func (q *RetryingQuoter) QuoteSellETH(ctx context.Context, ethAmount *big.Int, blockNumber *big.Int) (*uniswap.Quote, error) {
	var quote *uniswap.Quote
	err := Do(ctx, q.cfg, DefaultRetryable, func(ctx context.Context) error {
		var err error
		quote, err = q.inner.QuoteSellETH(ctx, ethAmount, blockNumber)
		return err
	})
	if err != nil && q.log != nil {
		q.log.Warn("quote sell eth failed after retries", "error", err)
	}
	return quote, err
}

// QuoteBuyETH retries QuoteBuyETH on transient failures.
func (q *RetryingQuoter) QuoteBuyETH(ctx context.Context, ethAmount *big.Int, blockNumber *big.Int) (*uniswap.Quote, error) {
	var quote *uniswap.Quote
	err := Do(ctx, q.cfg, DefaultRetryable, func(ctx context.Context) error {
		var err error
		quote, err = q.inner.QuoteBuyETH(ctx, ethAmount, blockNumber)
		return err
	})
	if err != nil && q.log != nil {
		q.log.Warn("quote buy eth failed after retries", "error", err)
	}
	return quote, err
}

var _ uniswap.Quoter = (*RetryingQuoter)(nil)
