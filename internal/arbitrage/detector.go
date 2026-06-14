package arbitrage

import (
	"github.com/shopspring/decimal"
)

// Detector evaluates CEX vs DEX prices and finds profitable arbitrage paths.
// It has no I/O — all prices are passed in via AnalysisInput.
type Detector struct{}

// NewDetector creates an arbitrage detector.
func NewDetector() *Detector {
	return &Detector{}
}

// Analyze checks both directions and returns the best opportunity above threshold, or nil.
func (d *Detector) Analyze(in AnalysisInput) *Opportunity {
	cexToDex := d.evaluateCEXToDEX(in)
	dexToCex := d.evaluateDEXToCEX(in)
	return pickBest(cexToDex, dexToCex, in.MinProfitPct)
}

func (d *Detector) evaluateCEXToDEX(in AnalysisInput) *Opportunity {
	// Buy ETH on Binance (asks + taker fee), sell ETH on Uniswap (WETH→USDC).
	costCEX := in.TradeSizeETH.Mul(in.CEXBuyUSD).Mul(decimalOne.Add(in.BinanceTakerFee))
	revenueDEX := in.TradeSizeETH.Mul(in.DEXSellUSD)
	profitUSD := revenueDEX.Sub(costCEX).Sub(in.GasCostUSD)

	return buildOpportunity(in, CEXToDEX, costCEX, profitUSD)
}

func (d *Detector) evaluateDEXToCEX(in AnalysisInput) *Opportunity {
	// Buy ETH on Uniswap (USDC→WETH), sell ETH on Binance (bids - taker fee).
	costDEX := in.TradeSizeETH.Mul(in.DEXBuyUSD)
	revenueCEX := in.TradeSizeETH.Mul(in.CEXSellUSD).Mul(decimalOne.Sub(in.BinanceTakerFee))
	profitUSD := revenueCEX.Sub(costDEX).Sub(in.GasCostUSD)

	return buildOpportunity(in, DEXToCEX, costDEX, profitUSD)
}

func buildOpportunity(
	in AnalysisInput,
	direction Direction,
	totalCost decimal.Decimal,
	profitUSD decimal.Decimal,
) *Opportunity {
	var cexPrice, dexPrice, priceDiffUSD decimal.Decimal

	switch direction {
	case CEXToDEX:
		cexPrice = in.CEXBuyUSD
		dexPrice = in.DEXSellUSD
		priceDiffUSD = dexPrice.Sub(cexPrice)
	case DEXToCEX:
		cexPrice = in.CEXSellUSD
		dexPrice = in.DEXBuyUSD
		priceDiffUSD = cexPrice.Sub(dexPrice)
	}

	priceDiffPct := decimal.Zero
	if cexPrice.IsPositive() {
		priceDiffPct = priceDiffUSD.Div(cexPrice).Mul(decimal100)
	}

	profitPct := decimal.Zero
	if totalCost.IsPositive() {
		profitPct = profitUSD.Div(totalCost).Mul(decimal100)
	}

	return &Opportunity{
		BlockNumber:  in.BlockNumber,
		Timestamp:    in.Timestamp,
		Direction:    direction,
		TradeSizeETH: in.TradeSizeETH,
		CEXPriceUSD:  cexPrice,
		DEXPriceUSD:  dexPrice,
		PriceDiffUSD: priceDiffUSD,
		PriceDiffPct: priceDiffPct,
		ProfitUSD:    profitUSD,
		GasCostUSD:   in.GasCostUSD,
		ProfitPct:    profitPct,
	}
}

func pickBest(a, b *Opportunity, minProfitPct decimal.Decimal) *Opportunity {
	var best *Opportunity

	if isProfitable(a, minProfitPct) {
		best = a
	}
	if isProfitable(b, minProfitPct) {
		if best == nil || b.ProfitUSD.GreaterThan(best.ProfitUSD) {
			best = b
		}
	}

	return best
}

func isProfitable(opp *Opportunity, minProfitPct decimal.Decimal) bool {
	if opp == nil {
		return false
	}
	return opp.ProfitUSD.GreaterThan(decimal.Zero) && opp.ProfitPct.GreaterThanOrEqual(minProfitPct)
}

var (
	decimalOne  = decimal.NewFromInt(1)
	decimal100  = decimal.NewFromInt(100)
)
