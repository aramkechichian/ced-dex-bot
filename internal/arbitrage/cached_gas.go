package arbitrage

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/aramik/ced-dex-bot/internal/cache"
	"github.com/shopspring/decimal"
)

const gasPriceCacheKey = "suggest_gas_price_wei"

// CachedGasEstimator wraps a GasEstimator and caches suggested gas price with TTL.
type CachedGasEstimator struct {
	inner GasCostEstimator
	rpc   gasPriceFetcher
	cache *cache.MemoryCache
	ttl   time.Duration
	log   *slog.Logger
}

type gasPriceFetcher interface {
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
}

// NewCachedGasEstimator caches RPC gas price lookups (~1 block TTL).
func NewCachedGasEstimator(
	inner *GasEstimator,
	c *cache.MemoryCache,
	ttl time.Duration,
	log *slog.Logger,
) *CachedGasEstimator {
	if ttl <= 0 {
		ttl = 12 * time.Second
	}
	return &CachedGasEstimator{
		inner: inner,
		rpc:   inner.rpc,
		cache: c,
		ttl:   ttl,
		log:   log,
	}
}

// SwapCostUSD uses cached gas price when available.
func (c *CachedGasEstimator) SwapCostUSD(ctx context.Context, ethPriceUSD decimal.Decimal) (decimal.Decimal, error) {
	if c == nil || c.inner == nil {
		return decimal.Zero, fmt.Errorf("cached gas estimator not configured")
	}

	gasPrice, hit, err := c.gasPrice(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	if hit && c.log != nil {
		c.log.Info("gas price cache hit", "ttl", c.ttl.String())
	}

	return swapCostFromGasPrice(gasPrice, ethPriceUSD), nil
}

func (c *CachedGasEstimator) gasPrice(ctx context.Context) (*big.Int, bool, error) {
	if v, ok := c.cache.Get(gasPriceCacheKey); ok {
		if gasPrice, ok := v.(*big.Int); ok && gasPrice != nil {
			return gasPrice, true, nil
		}
	}

	if c.rpc == nil {
		// Fallback to inner (uncached) if rpc not wired.
		return nil, false, fmt.Errorf("gas price fetcher not configured")
	}

	gasPrice, err := c.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("suggest gas price: %w", err)
	}

	c.cache.Set(gasPriceCacheKey, new(big.Int).Set(gasPrice), c.ttl)
	return gasPrice, false, nil
}

var _ GasCostEstimator = (*CachedGasEstimator)(nil)
