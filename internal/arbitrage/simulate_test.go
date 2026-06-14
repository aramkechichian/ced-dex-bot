package arbitrage

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/shopspring/decimal"
)

func TestRunSimulateDetectsOpportunity(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := ServiceConfig{
		TradeSizesETH:   []decimal.Decimal{decimal.NewFromInt(10)},
		BinanceTakerFee: decimal.RequireFromString("0.001"),
		MinProfitPct:    decimal.RequireFromString("0.5"),
		Pretty:          false,
	}

	if err := RunSimulate(context.Background(), cfg, NewDetector(), NewOpportunityEmitter(nil, false), log); err != nil {
		t.Fatal(err)
	}
}

func TestOpportunityEmitterPrintOnly(t *testing.T) {
	emit := NewOpportunityEmitter(nil, false)
	opp := &Opportunity{
		BlockNumber:  1,
		Direction:    CEXToDEX,
		TradeSizeETH: decimal.NewFromInt(10),
		CEXPriceUSD:  decimal.NewFromInt(100),
		DEXPriceUSD:  decimal.NewFromInt(110),
		ProfitUSD:    decimal.NewFromInt(50),
		ProfitPct:    decimal.NewFromFloat(1),
		GasCostUSD:   decimal.NewFromInt(1),
	}
	if err := emit.Emit(context.Background(), opp, true); err != nil {
		t.Fatal(err)
	}
}
