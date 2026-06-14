package binance

import (
	"testing"

	"github.com/shopspring/decimal"
)

func sampleOrderbook() *Orderbook {
	return &Orderbook{
		Symbol: "ETHUSDC",
		Bids: []Level{
			{Price: decimal.RequireFromString("2245.30"), Quantity: decimal.RequireFromString("1.5")},
			{Price: decimal.RequireFromString("2245.20"), Quantity: decimal.RequireFromString("2.3")},
			{Price: decimal.RequireFromString("2245.10"), Quantity: decimal.RequireFromString("10.0")},
		},
		Asks: []Level{
			{Price: decimal.RequireFromString("2245.50"), Quantity: decimal.RequireFromString("1.2")},
			{Price: decimal.RequireFromString("2245.60"), Quantity: decimal.RequireFromString("2.3")},
			{Price: decimal.RequireFromString("2245.70"), Quantity: decimal.RequireFromString("10.0")},
		},
	}
}

func TestEffectiveBuyPriceSingleLevel(t *testing.T) {
	svc := NewService()
	book := sampleOrderbook()

	price, err := svc.EffectiveBuyPrice(book, decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !price.Equal(decimal.RequireFromString("2245.50")) {
		t.Fatalf("buy price: got %s want 2245.50", price)
	}
}

func TestEffectiveBuyPriceWalksAsks(t *testing.T) {
	svc := NewService()
	book := sampleOrderbook()

	// 1.2 @ 2245.50 + 0.8 @ 2245.60 = 2694.60 + 1796.48 = 4491.08 / 2 = 2245.54
	price, err := svc.EffectiveBuyPrice(book, decimal.NewFromInt(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := decimal.RequireFromString("2245.54")
	if !price.Equal(want) {
		t.Fatalf("buy price: got %s want %s", price, want)
	}
}

func TestEffectiveSellPriceWalksBids(t *testing.T) {
	svc := NewService()
	book := sampleOrderbook()

	// 1.5 @ 2245.30 + 0.5 @ 2245.20 = 3367.95 + 1122.60 = 4490.55 / 2 = 2245.275
	price, err := svc.EffectiveSellPrice(book, decimal.NewFromInt(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := decimal.RequireFromString("2245.275")
	if !price.Equal(want) {
		t.Fatalf("sell price: got %s want %s", price, want)
	}
}

func TestEffectivePricesTenETH(t *testing.T) {
	svc := NewService()
	book := sampleOrderbook()

	prices, err := svc.EffectivePrices(book, decimal.NewFromInt(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Buy 10 ETH: 1.2*2245.50 + 2.3*2245.60 + 6.5*2245.70 = 22456.53 / 10 = 2245.653
	if !prices.EffectiveBuyUSD.Equal(decimal.RequireFromString("2245.653")) {
		t.Fatalf("buy: got %s", prices.EffectiveBuyUSD)
	}

	// Sell 10 ETH: 1.5*2245.30 + 2.3*2245.20 + 6.2*2245.10 = 22451.53 / 10 = 2245.153
	if !prices.EffectiveSellUSD.Equal(decimal.RequireFromString("2245.153")) {
		t.Fatalf("sell: got %s", prices.EffectiveSellUSD)
	}
}

func TestEffectiveBuyPriceInsufficientLiquidity(t *testing.T) {
	svc := NewService()
	book := sampleOrderbook()

	_, err := svc.EffectiveBuyPrice(book, decimal.NewFromInt(100))
	if err == nil {
		t.Fatal("expected insufficient liquidity error")
	}
}

func TestEffectivePricesInvalidBook(t *testing.T) {
	svc := NewService()
	_, err := svc.EffectivePrices(nil, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected error for nil book")
	}
}
