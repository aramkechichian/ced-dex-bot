package arbitrage

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// Example inspired by the challenge PDF (CEX cheaper, DEX more expensive → CEX→DEX).
func TestAnalyzeCEXToDEXOpportunity(t *testing.T) {
	d := NewDetector()

	in := AnalysisInput{
		BlockNumber:     18234567,
		Timestamp:       time.Date(2024, 1, 15, 14, 23, 45, 0, time.UTC),
		TradeSizeETH:    decimal.NewFromInt(10),
		CEXBuyUSD:       decimal.RequireFromString("2245.30"),
		CEXSellUSD:      decimal.RequireFromString("2240.00"),
		DEXSellUSD:      decimal.RequireFromString("2267.80"),
		DEXBuyUSD:       decimal.RequireFromString("2270.00"),
		GasCostUSD:      decimal.NewFromFloat(9.0),
		BinanceTakerFee: decimal.RequireFromString("0.001"),
		MinProfitPct:    decimal.RequireFromString("0.5"),
	}

	opp := d.Analyze(in)
	if opp == nil {
		t.Fatal("expected opportunity")
	}
	if opp.Direction != CEXToDEX {
		t.Fatalf("direction: %s", opp.Direction)
	}
	if !opp.ProfitUSD.GreaterThan(decimal.Zero) {
		t.Fatalf("expected positive profit, got %s", opp.ProfitUSD)
	}
	if opp.TradeSizeETH.String() != "10" {
		t.Fatalf("trade size: %s", opp.TradeSizeETH)
	}
}

func TestAnalyzeDEXToCEXOpportunity(t *testing.T) {
	d := NewDetector()

	in := AnalysisInput{
		BlockNumber:     1,
		TradeSizeETH:    decimal.NewFromInt(10),
		CEXBuyUSD:       decimal.RequireFromString("2300.00"),
		CEXSellUSD:      decimal.RequireFromString("2280.00"),
		DEXSellUSD:      decimal.RequireFromString("2260.00"),
		DEXBuyUSD:       decimal.RequireFromString("2250.00"),
		GasCostUSD:      decimal.NewFromFloat(9.0),
		BinanceTakerFee: decimal.RequireFromString("0.001"),
		MinProfitPct:    decimal.RequireFromString("0.5"),
	}

	opp := d.Analyze(in)
	if opp == nil {
		t.Fatal("expected opportunity")
	}
	if opp.Direction != DEXToCEX {
		t.Fatalf("direction: %s", opp.Direction)
	}
}

func TestAnalyzeNoOpportunityWhenSpreadTooSmall(t *testing.T) {
	d := NewDetector()

	in := AnalysisInput{
		TradeSizeETH:    decimal.NewFromInt(1),
		CEXBuyUSD:       decimal.RequireFromString("2000.00"),
		CEXSellUSD:      decimal.RequireFromString("1999.50"),
		DEXSellUSD:      decimal.RequireFromString("2000.10"),
		DEXBuyUSD:       decimal.RequireFromString("2000.20"),
		GasCostUSD:      decimal.NewFromFloat(9.0),
		BinanceTakerFee: decimal.RequireFromString("0.001"),
		MinProfitPct:    decimal.RequireFromString("0.5"),
	}

	if opp := d.Analyze(in); opp != nil {
		t.Fatalf("expected nil, got profit %s", opp.ProfitUSD)
	}
}

func TestAnalyzeGasEatsProfit(t *testing.T) {
	d := NewDetector()

	in := AnalysisInput{
		TradeSizeETH:    decimal.NewFromInt(1),
		CEXBuyUSD:       decimal.RequireFromString("2000.00"),
		CEXSellUSD:      decimal.RequireFromString("1999.00"),
		DEXSellUSD:      decimal.RequireFromString("2010.00"),
		DEXBuyUSD:       decimal.RequireFromString("2012.00"),
		GasCostUSD:      decimal.NewFromFloat(50.0),
		BinanceTakerFee: decimal.RequireFromString("0.001"),
		MinProfitPct:    decimal.RequireFromString("0.1"),
	}

	if opp := d.Analyze(in); opp != nil {
		t.Fatalf("gas should eliminate profit, got %s", opp.ProfitUSD)
	}
}

func TestFormatOpportunity(t *testing.T) {
	opp := &Opportunity{
		BlockNumber:  18234567,
		Timestamp:    time.Date(2024, 1, 15, 14, 23, 45, 0, time.UTC),
		Direction:    CEXToDEX,
		TradeSizeETH: decimal.NewFromInt(10),
		CEXPriceUSD:  decimal.RequireFromString("2245.30"),
		DEXPriceUSD:  decimal.RequireFromString("2267.80"),
		PriceDiffUSD: decimal.RequireFromString("22.50"),
		PriceDiffPct: decimal.RequireFromString("1.00"),
		ProfitUSD:    decimal.RequireFromString("221.00"),
		GasCostUSD:   decimal.NewFromFloat(9.0),
		ProfitPct:    decimal.RequireFromString("0.98"),
	}

	out := FormatOpportunity(opp)
	if !strings.Contains(out, "ARBITRAGE OPPORTUNITY DETECTED") {
		t.Fatal("missing header")
	}
	if !strings.Contains(out, "CEX → DEX") {
		t.Fatal("missing direction")
	}
}

func TestCEXToDEXProfitCalculation(t *testing.T) {
	d := NewDetector()
	in := AnalysisInput{
		TradeSizeETH:    decimal.NewFromInt(10),
		CEXBuyUSD:       decimal.RequireFromString("2245.30"),
		CEXSellUSD:      decimal.RequireFromString("2240.00"),
		DEXSellUSD:      decimal.RequireFromString("2267.80"),
		DEXBuyUSD:       decimal.RequireFromString("2270.00"),
		GasCostUSD:      decimal.Zero,
		BinanceTakerFee: decimal.RequireFromString("0.001"),
		MinProfitPct:    decimal.Zero,
	}

	opp := d.Analyze(in)
	if opp == nil {
		t.Fatal("expected opportunity")
	}

	// revenue 22678 - cost (22453 * 1.001) ≈ 202.55 after Binance taker fee
	expected := decimal.RequireFromString("202.547")
	if !opp.ProfitUSD.Equal(expected) {
		t.Fatalf("profit: got %s want %s", opp.ProfitUSD, expected)
	}
}
