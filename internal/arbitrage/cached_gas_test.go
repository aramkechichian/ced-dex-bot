package arbitrage

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/aramik/ced-dex-bot/internal/cache"
	"github.com/shopspring/decimal"
)

type fakeGasPriceRPC struct {
	calls int
	price *big.Int
}

func (f *fakeGasPriceRPC) SuggestGasPrice(_ context.Context) (*big.Int, error) {
	f.calls++
	return new(big.Int).Set(f.price), nil
}

func TestCachedGasEstimatorUsesCache(t *testing.T) {
	rpc := &fakeGasPriceRPC{price: big.NewInt(1_000_000_000)}
	inner := &GasEstimator{rpc: nil}
	cached := &CachedGasEstimator{
		inner: inner,
		rpc:   rpc,
		cache: cache.NewMemoryCache(),
		ttl:   time.Minute,
		log:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	ethPrice := decimal.NewFromInt(2000)

	if _, err := cached.SwapCostUSD(context.Background(), ethPrice); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := cached.SwapCostUSD(context.Background(), ethPrice); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if rpc.calls != 1 {
		t.Fatalf("expected 1 rpc call, got %d", rpc.calls)
	}
}
