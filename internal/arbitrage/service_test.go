package arbitrage

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/aramik/ced-dex-bot/internal/binance"
	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/aramik/ced-dex-bot/internal/uniswap"
	"github.com/shopspring/decimal"
)

type fakeExchange struct {
	book *binance.Orderbook
	err  error
}

func (f *fakeExchange) GetOrderbook(_ context.Context, _ string) (*binance.Orderbook, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.book, nil
}

type fakeQuoter struct {
	sellUSD decimal.Decimal
	buyUSD  decimal.Decimal
}

func (f *fakeQuoter) QuoteSellETH(_ context.Context, _ *big.Int, blockNumber *big.Int) (*uniswap.Quote, error) {
	return &uniswap.Quote{
		Direction:      uniswap.WETHToUSDC,
		EffectivePrice: f.sellUSD,
		BlockNumber:    blockNumber.Uint64(),
	}, nil
}

func (f *fakeQuoter) QuoteBuyETH(_ context.Context, _ *big.Int, blockNumber *big.Int) (*uniswap.Quote, error) {
	return &uniswap.Quote{
		Direction:      uniswap.USDCToWETH,
		EffectivePrice: f.buyUSD,
		BlockNumber:    blockNumber.Uint64(),
	}, nil
}

type fakeGas struct {
	cost decimal.Decimal
}

func (f *fakeGas) SwapCostUSD(_ context.Context, _ decimal.Decimal) (decimal.Decimal, error) {
	return f.cost, nil
}

func TestServiceShouldProcessDedup(t *testing.T) {
	s := &Service{}
	if !s.shouldProcess(100) {
		t.Fatal("expected first block")
	}
	if s.shouldProcess(100) {
		t.Fatal("duplicate block should be skipped")
	}
	if !s.shouldProcess(101) {
		t.Fatal("expected next block")
	}
}

func TestServiceAnalyzeSizeDetectsOpportunity(t *testing.T) {
	book := &binance.Orderbook{
		Symbol: "ETHUSDC",
		Bids: []binance.Level{
			{Price: decimal.RequireFromString("2240.00"), Quantity: decimal.NewFromInt(1000)},
		},
		Asks: []binance.Level{
			{Price: decimal.RequireFromString("2245.30"), Quantity: decimal.NewFromInt(1000)},
		},
	}

	svc := NewService(
		ServiceConfig{
			Symbol:          "ETHUSDC",
			TradeSizesETH:   []decimal.Decimal{decimal.NewFromInt(10)},
			BinanceTakerFee: decimal.RequireFromString("0.001"),
			MinProfitPct:    decimal.RequireFromString("0.5"),
		},
		nil,
		&fakeExchange{book: book},
		binance.NewService(),
		&fakeQuoter{
			sellUSD: decimal.RequireFromString("2267.80"),
			buyUSD:  decimal.RequireFromString("2270.00"),
		},
		NewDetector(),
		&fakeGas{cost: decimal.NewFromFloat(9)},
		slog.New(slog.NewTextHandler(os.Stderr, nil)),
	)

	eval, err := svc.evaluateSize(
		context.Background(),
		ethereum.Block{Number: 42, Timestamp: time.Now().UTC()},
		book,
		big.NewInt(42),
		decimal.NewFromInt(10),
		decimal.NewFromFloat(9),
	)
	if err != nil {
		t.Fatalf("evaluateSize: %v", err)
	}
	if eval.Opportunity == nil {
		t.Fatal("expected opportunity")
	}
	if eval.Opportunity.Direction != CEXToDEX {
		t.Fatalf("direction: %s", eval.Opportunity.Direction)
	}
	if eval.BestEffort == nil {
		t.Fatal("expected best effort")
	}
}

func TestServiceProcessBlockEndToEnd(t *testing.T) {
	book := &binance.Orderbook{
		Symbol: "ETHUSDC",
		Bids: []binance.Level{
			{Price: decimal.RequireFromString("2240.00"), Quantity: decimal.NewFromInt(1000)},
		},
		Asks: []binance.Level{
			{Price: decimal.RequireFromString("2245.30"), Quantity: decimal.NewFromInt(1000)},
		},
	}

	svc := NewService(
		ServiceConfig{
			Symbol:          "ETHUSDC",
			TradeSizesETH:   []decimal.Decimal{decimal.NewFromInt(10)},
			BinanceTakerFee: decimal.RequireFromString("0.001"),
			MinProfitPct:    decimal.RequireFromString("0.5"),
			MaxBlocks:       1,
		},
		nil,
		&fakeExchange{book: book},
		binance.NewService(),
		&fakeQuoter{
			sellUSD: decimal.RequireFromString("2267.80"),
			buyUSD:  decimal.RequireFromString("2270.00"),
		},
		NewDetector(),
		&fakeGas{cost: decimal.NewFromFloat(9)},
		slog.New(slog.NewTextHandler(os.Stderr, nil)),
	)

	block := ethereum.Block{
		Number:    100,
		Hash:      "0xabc",
		Timestamp: time.Now().UTC(),
	}

	if err := svc.processBlock(context.Background(), block); err != nil {
		t.Fatalf("processBlock: %v", err)
	}
	if svc.processed != 1 {
		t.Fatalf("processed: %d", svc.processed)
	}
}

func TestServiceProcessBlockPrettyMode(t *testing.T) {
	book := &binance.Orderbook{
		Symbol: "ETHUSDC",
		Bids: []binance.Level{
			{Price: decimal.RequireFromString("2000.00"), Quantity: decimal.NewFromInt(1000)},
		},
		Asks: []binance.Level{
			{Price: decimal.RequireFromString("2010.00"), Quantity: decimal.NewFromInt(1000)},
		},
	}

	svc := NewService(
		ServiceConfig{
			Symbol:          "ETHUSDC",
			TradeSizesETH:   []decimal.Decimal{decimal.NewFromInt(1)},
			BinanceTakerFee: decimal.RequireFromString("0.001"),
			MinProfitPct:    decimal.RequireFromString("0.5"),
			Pretty:          true,
		},
		nil,
		&fakeExchange{book: book},
		binance.NewService(),
		&fakeQuoter{
			sellUSD: decimal.RequireFromString("2005.00"),
			buyUSD:  decimal.RequireFromString("2006.00"),
		},
		NewDetector(),
		&fakeGas{cost: decimal.NewFromFloat(1)},
		slog.New(slog.NewTextHandler(os.Stderr, nil)),
	)

	block := ethereum.Block{
		Number:    200,
		Timestamp: time.Now().UTC(),
	}

	if err := svc.processBlock(context.Background(), block); err != nil {
		t.Fatalf("processBlock: %v", err)
	}
}

func TestMidPrice(t *testing.T) {
	book := &binance.Orderbook{
		Bids: []binance.Level{{Price: decimal.NewFromInt(2000), Quantity: decimal.NewFromInt(1)}},
		Asks: []binance.Level{{Price: decimal.NewFromInt(2010), Quantity: decimal.NewFromInt(1)}},
	}
	mid, err := midPrice(book)
	if err != nil {
		t.Fatal(err)
	}
	if !mid.Equal(decimal.NewFromInt(2005)) {
		t.Fatalf("mid: %s", mid)
	}
}

func TestWeiToETH(t *testing.T) {
	oneETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	got := weiToETH(oneETH)
	if !got.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("weiToETH: %s", got)
	}
}
