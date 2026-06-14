package arbitrage

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// FormatOpportunity renders the challenge-style arbitrage alert.
func FormatOpportunity(opp *Opportunity) string {
	if opp == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("=== ARBITRAGE OPPORTUNITY DETECTED ===\n")
	fmt.Fprintf(&b, "Block Number: %d\n", opp.BlockNumber)
	fmt.Fprintf(&b, "Timestamp: %s\n", opp.Timestamp.UTC().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(&b, "Direction: %s\n", opp.Direction)
	fmt.Fprintf(&b, "Trade Size: %s ETH\n", opp.TradeSizeETH.StringFixed(1))
	fmt.Fprintf(&b, "CEX Price: $%s (effective with slippage)\n", opp.CEXPriceUSD.StringFixed(2))
	fmt.Fprintf(&b, "DEX Price: $%s (effective with slippage)\n", opp.DEXPriceUSD.StringFixed(2))
	fmt.Fprintf(&b, "Price Difference: $%s per ETH (%s%%)\n",
		opp.PriceDiffUSD.StringFixed(2),
		opp.PriceDiffPct.StringFixed(2),
	)
	fmt.Fprintf(&b, "Estimated Profit: $%s (before gas and fees)\n", grossProfitUSD(opp).StringFixed(2))
	fmt.Fprintf(&b, "Gas Cost: $%s\n", opp.GasCostUSD.StringFixed(2))
	fmt.Fprintf(&b, "Net Profit: $%s (after gas and fees)\n", opp.ProfitUSD.StringFixed(2))
	fmt.Fprintf(&b, "Return on Capital: %s%%\n", opp.ProfitPct.StringFixed(2))
	b.WriteString(formatExecutionSteps(opp))

	return b.String()
}

// grossProfitUSD is the price-spread profit before gas and CEX taker fee adjustments
// (matches the challenge PDF "Estimated Profit before gas and fees").
func grossProfitUSD(opp *Opportunity) decimal.Decimal {
	if opp == nil {
		return decimal.Zero
	}
	return opp.PriceDiffUSD.Mul(opp.TradeSizeETH)
}

func formatExecutionSteps(opp *Opportunity) string {
	size := opp.TradeSizeETH.StringFixed(1)
	cexTotal := opp.TradeSizeETH.Mul(opp.CEXPriceUSD)
	dexTotal := opp.TradeSizeETH.Mul(opp.DEXPriceUSD)

	switch opp.Direction {
	case CEXToDEX:
		return fmt.Sprintf(`Execution Steps:
1. Buy %s ETH on Binance at average price $%s
   - Required capital: ~$%s USDC
2. Transfer ETH to trading wallet
3. Execute Uniswap V3 swap: %s ETH → USDC
   - Expected output: ~$%s USDC (after 0.3%% pool fee)
`, size, opp.CEXPriceUSD.StringFixed(2), cexTotal.StringFixed(2),
			size, dexTotal.StringFixed(2))

	case DEXToCEX:
		return fmt.Sprintf(`Execution Steps:
1. Buy %s ETH on Uniswap V3 with ~$%s USDC
2. Transfer ETH to Binance
3. Sell %s ETH on Binance at average price $%s
   - Expected output: ~$%s USDC
`, size, dexTotal.StringFixed(2), size, opp.CEXPriceUSD.StringFixed(2), cexTotal.StringFixed(2))

	default:
		return ""
	}
}

// ProfitSummary returns a one-line summary for logging.
func ProfitSummary(opp *Opportunity) string {
	if opp == nil {
		return "no opportunity"
	}
	return fmt.Sprintf("%s profit=$%s (%s%%)",
		opp.Direction,
		opp.ProfitUSD.StringFixed(2),
		opp.ProfitPct.StringFixed(2),
	)
}
