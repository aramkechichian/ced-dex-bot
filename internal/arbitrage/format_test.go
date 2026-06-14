package arbitrage

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestFormatOpportunityMatchesChallengeLayout(t *testing.T) {
	opp := &Opportunity{
		BlockNumber:  18234567,
		Timestamp:    time.Date(2024, 1, 15, 14, 23, 45, 0, time.UTC),
		Direction:    CEXToDEX,
		TradeSizeETH: decimal.NewFromInt(10),
		CEXPriceUSD:  decimal.RequireFromString("2245.30"),
		DEXPriceUSD:  decimal.RequireFromString("2267.80"),
		PriceDiffUSD: decimal.RequireFromString("22.50"),
		PriceDiffPct: decimal.RequireFromString("1.00"),
		ProfitUSD:    decimal.RequireFromString("193.55"),
		GasCostUSD:   decimal.NewFromFloat(9.0),
		ProfitPct:    decimal.RequireFromString("0.86"),
	}

	out := FormatOpportunity(opp)

	checks := []string{
		"=== ARBITRAGE OPPORTUNITY DETECTED ===",
		"Block Number: 18234567",
		"Timestamp: 2024-01-15 14:23:45 UTC",
		"Direction: CEX → DEX (Buy on Binance, Sell on Uniswap)",
		"Trade Size: 10.0 ETH",
		"CEX Price: $2245.30 (effective with slippage)",
		"DEX Price: $2267.80 (effective with slippage)",
		"Price Difference: $22.50 per ETH (1.00%)",
		"Estimated Profit: $225.00 (before gas and fees)",
		"Gas Cost: $9.00",
		"Net Profit: $193.55 (after gas and fees)",
		"Execution Steps:",
		"Buy 10.0 ETH on Binance",
		"Execute Uniswap V3 swap: 10.0 ETH → USDC",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestGrossProfitUSD(t *testing.T) {
	opp := &Opportunity{
		PriceDiffUSD: decimal.RequireFromString("22.50"),
		TradeSizeETH: decimal.NewFromInt(10),
	}
	got := grossProfitUSD(opp)
	want := decimal.RequireFromString("225.00")
	if !got.Equal(want) {
		t.Fatalf("gross: %s want %s", got, want)
	}
}
