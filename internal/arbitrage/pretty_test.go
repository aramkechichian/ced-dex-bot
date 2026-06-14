package arbitrage

import (
	"strings"
	"testing"
	"time"

	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/shopspring/decimal"
)

func TestFormatBlockTable(t *testing.T) {
	block := ethereum.Block{
		Number:    12345678,
		Timestamp: time.Date(2024, 1, 15, 14, 23, 45, 0, time.UTC),
	}

	eval := SizeEvaluation{
		Input: AnalysisInput{
			TradeSizeETH: decimal.NewFromInt(10),
			CEXBuyUSD:    decimal.RequireFromString("2245.30"),
			CEXSellUSD:   decimal.RequireFromString("2240.00"),
			DEXSellUSD:   decimal.RequireFromString("2267.80"),
			DEXBuyUSD:    decimal.RequireFromString("2270.00"),
			MinProfitPct: decimal.RequireFromString("0.5"),
		},
		CEXToDEX:    &Opportunity{ProfitUSD: decimal.RequireFromString("193.55"), ProfitPct: decimal.RequireFromString("0.86"), Direction: CEXToDEX, TradeSizeETH: decimal.NewFromInt(10)},
		DEXToCEX:    &Opportunity{ProfitUSD: decimal.RequireFromString("-50.00"), ProfitPct: decimal.RequireFromString("-0.22"), Direction: DEXToCEX, TradeSizeETH: decimal.NewFromInt(10)},
		BestEffort:  &Opportunity{ProfitUSD: decimal.RequireFromString("193.55"), ProfitPct: decimal.RequireFromString("0.86"), Direction: CEXToDEX, TradeSizeETH: decimal.NewFromInt(10)},
		Opportunity: &Opportunity{ProfitUSD: decimal.RequireFromString("193.55"), ProfitPct: decimal.RequireFromString("0.86"), Direction: CEXToDEX, TradeSizeETH: decimal.NewFromInt(10)},
	}

	out := FormatBlockTable(BlockReport{
		Block:         block,
		EthMidUSD:     decimal.RequireFromString("2242.65"),
		GasUSD:        decimal.NewFromFloat(9),
		Evaluations:   []SizeEvaluation{eval},
		BlockBest:     eval.BestEffort,
		Opportunities: 1,
		MinProfitPct:  decimal.RequireFromString("0.5"),
		Duration:      120 * time.Millisecond,
	})

	checks := []string{
		"Block 12345678",
		"ETH mid $2242.65",
		"Gas cost $9.00",
		"2245.30",
		"2267.80",
		"CEX→DEX",
		"opportunities: 1",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}
