package arbitrage

import (
	"time"

	"github.com/shopspring/decimal"
)

// Direction is the profitable arbitrage path between CEX and DEX.
type Direction int

const (
	// CEXToDEX: buy ETH on Binance, sell on Uniswap.
	CEXToDEX Direction = iota
	// DEXToCEX: buy ETH on Uniswap, sell on Binance.
	DEXToCEX
)

func (d Direction) String() string {
	switch d {
	case CEXToDEX:
		return "CEX → DEX (Buy on Binance, Sell on Uniswap)"
	case DEXToCEX:
		return "DEX → CEX (Buy on Uniswap, Sell on Binance)"
	default:
		return "unknown"
	}
}

// AnalysisInput bundles all prices needed by the detector (pure logic, no I/O).
type AnalysisInput struct {
	BlockNumber     uint64
	Timestamp       time.Time
	TradeSizeETH    decimal.Decimal
	CEXBuyUSD       decimal.Decimal // effective buy price on Binance
	CEXSellUSD      decimal.Decimal // effective sell price on Binance
	DEXSellUSD      decimal.Decimal // effective price selling ETH on Uniswap
	DEXBuyUSD       decimal.Decimal // effective price buying ETH on Uniswap
	GasCostUSD      decimal.Decimal
	BinanceTakerFee decimal.Decimal
	MinProfitPct    decimal.Decimal
}

// Opportunity is a detected arbitrage signal ready to print or log.
type Opportunity struct {
	BlockNumber  uint64
	Timestamp    time.Time
	Direction    Direction
	TradeSizeETH decimal.Decimal
	CEXPriceUSD  decimal.Decimal // effective with slippage
	DEXPriceUSD  decimal.Decimal
	PriceDiffUSD decimal.Decimal // per ETH
	PriceDiffPct decimal.Decimal
	ProfitUSD    decimal.Decimal // net after gas and fees
	GasCostUSD   decimal.Decimal
	ProfitPct    decimal.Decimal
}
