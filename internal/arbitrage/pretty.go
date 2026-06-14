package arbitrage

import (
	"fmt"
	"strings"
	"time"

	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/shopspring/decimal"
)

// BlockReport aggregates per-block analysis for pretty printing.
type BlockReport struct {
	Block         ethereum.Block
	EthMidUSD     decimal.Decimal
	GasUSD        decimal.Decimal
	Evaluations   []SizeEvaluation
	BlockBest     *Opportunity
	Opportunities int
	MinProfitPct  decimal.Decimal
	Duration      time.Duration
	Simulated     bool
}

// FormatBlockTable renders a human-readable ASCII table for one block.
func FormatBlockTable(r BlockReport) string {
	const width = 78

	var b strings.Builder
	top := strings.Repeat("═", width)
	rule := strings.Repeat("─", width)

	fmt.Fprintf(&b, "%s\n", top)
	if r.Simulated {
		fmt.Fprintf(&b, " [SIMULATED] Demo data from challenge PDF — no live market data\n")
		fmt.Fprintf(&b, "%s\n", top)
	}
	fmt.Fprintf(&b, " Block %-10d │ %s\n",
		r.Block.Number,
		r.Block.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC"),
	)
	fmt.Fprintf(&b, " ETH mid $%s │ Gas cost $%s (180k units, net in profit cols)\n",
		r.EthMidUSD.StringFixed(2),
		r.GasUSD.StringFixed(2),
	)
	fmt.Fprintf(&b, "%s\n", top)
	fmt.Fprintf(&b, " %5s │ %8s │ %8s │ %8s │ %8s │ %16s │ %16s\n",
		"Size", "CEX buy", "CEX sell", "DEX sell", "DEX buy", "CEX→DEX", "DEX→CEX",
	)
	fmt.Fprintf(&b, "%s\n", rule)

	for _, eval := range r.Evaluations {
		in := eval.Input
		fmt.Fprintf(&b, " %5s │ %8s │ %8s │ %8s │ %8s │ %16s │ %16s\n",
			in.TradeSizeETH.StringFixed(1),
			in.CEXBuyUSD.StringFixed(2),
			in.CEXSellUSD.StringFixed(2),
			in.DEXSellUSD.StringFixed(2),
			in.DEXBuyUSD.StringFixed(2),
			formatProfitCell(eval.CEXToDEX),
			formatProfitCell(eval.DEXToCEX),
		)
	}

	fmt.Fprintf(&b, "%s\n", rule)
	fmt.Fprintf(&b, " %s\n", formatBlockFooter(r))
	fmt.Fprintf(&b, "%s\n", top)

	return b.String()
}

func formatProfitCell(opp *Opportunity) string {
	if opp == nil {
		return "n/a"
	}
	return fmt.Sprintf("$%s (%s%%)", opp.ProfitUSD.StringFixed(2), opp.ProfitPct.StringFixed(2))
}

func formatBlockFooter(r BlockReport) string {
	bestDir := shortDirection(r.BlockBest)
	bestPct := formatProfitPct(r.BlockBest)
	bestSize := "n/a"
	if r.BlockBest != nil {
		bestSize = r.BlockBest.TradeSizeETH.StringFixed(1) + " ETH"
	}

	return fmt.Sprintf("Best: %s @ %s (%s%%) │ gas $%s │ threshold: %s%% │ opportunities: %d │ market: %s │ %dms",
		bestDir,
		bestSize,
		bestPct,
		r.GasUSD.StringFixed(2),
		r.MinProfitPct.StringFixed(2),
		r.Opportunities,
		marketStatus(r.BlockBest, r.MinProfitPct),
		r.Duration.Milliseconds(),
	)
}

func marketStatus(best *Opportunity, threshold decimal.Decimal) string {
	if best == nil {
		return "unknown"
	}
	if best.ProfitPct.GreaterThanOrEqual(threshold) && best.ProfitUSD.GreaterThan(decimal.Zero) {
		return "opportunity"
	}
	gap := threshold.Sub(best.ProfitPct)
	return fmt.Sprintf("aligned (%.2f%% below threshold)", gap.InexactFloat64())
}
