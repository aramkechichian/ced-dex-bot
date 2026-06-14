package resilience

import (
	"context"
	"fmt"

	"github.com/aramik/ced-dex-bot/internal/binance"
	"golang.org/x/time/rate"
)

// RateLimitedExchange wraps a CEX client with a token-bucket rate limiter.
type RateLimitedExchange struct {
	inner   binance.Exchange
	limiter *rate.Limiter
}

// NewRateLimitedExchange limits requests per second to protect API quotas.
func NewRateLimitedExchange(inner binance.Exchange, rps float64, burst int) *RateLimitedExchange {
	if rps <= 0 {
		rps = 5
	}
	if burst <= 0 {
		burst = 10
	}
	return &RateLimitedExchange{
		inner:   inner,
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// GetOrderbook waits for rate limit capacity before calling the inner exchange.
func (r *RateLimitedExchange) GetOrderbook(ctx context.Context, symbol string) (*binance.Orderbook, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}
	return r.inner.GetOrderbook(ctx, symbol)
}

var _ binance.Exchange = (*RateLimitedExchange)(nil)
