package binance

import (
	"time"

	"github.com/shopspring/decimal"
)

// Level is a single orderbook price level from Binance.
// Price is in USDC per ETH; Quantity is ETH available at that price.
type Level struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// Orderbook is a point-in-time snapshot of the Binance ETH-USDC book.
type Orderbook struct {
	Symbol       string
	Bids         []Level // buyers: best (highest) price first
	Asks         []Level // sellers: best (lowest) price first
	LastUpdateID int64
	FetchedAt    time.Time
}

// SidePrices holds effective execution prices for a given ETH trade size.
// Used by the arbitrage detector after walking the orderbook.
type SidePrices struct {
	TradeSizeETH     decimal.Decimal
	EffectiveBuyUSD  decimal.Decimal // USDC spent per ETH when buying (walk asks)
	EffectiveSellUSD decimal.Decimal // USDC received per ETH when selling (walk bids)
}
