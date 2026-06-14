package arbitrage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/shopspring/decimal"
)

// RunSimulate executes an offline demo using challenge PDF prices (no network).
func RunSimulate(
	ctx context.Context,
	cfg ServiceConfig,
	detector *Detector,
	emit OpportunityEmitter,
	log *slog.Logger,
) error {
	if detector == nil {
		return fmt.Errorf("detector is required")
	}

	block := simulatedBlock()
	gasUSD := decimal.NewFromFloat(9.0)
	ethMid := decimal.RequireFromString("2242.65")

	log.Info("simulate mode started",
		"trade_sizes", tradeSizesString(cfg.TradeSizesETH),
		"pretty", cfg.Pretty,
	)

	start := time.Now()
	opportunities := 0
	var blockBest *Opportunity
	var bestOpp *Opportunity
	evaluations := make([]SizeEvaluation, 0, len(cfg.TradeSizesETH))

	for _, size := range cfg.TradeSizesETH {
		input := simulatedInput(block, size, gasUSD, cfg.BinanceTakerFee, cfg.MinProfitPct)
		eval := detector.Evaluate(input)
		evaluations = append(evaluations, eval)

		if !cfg.Pretty {
			logSizeAnalysisFromEval(log, block.Number, eval)
		}

		if eval.BestEffort != nil {
			if blockBest == nil || eval.BestEffort.ProfitPct.GreaterThan(blockBest.ProfitPct) {
				blockBest = eval.BestEffort
			}
		}

		if eval.Opportunity != nil {
			opportunities++
			emit.Print(eval.Opportunity, true)
			if bestOpp == nil || eval.Opportunity.ProfitUSD.GreaterThan(bestOpp.ProfitUSD) {
				bestOpp = eval.Opportunity
			}
		}
	}

	if bestOpp != nil {
		if err := emit.Notify(ctx, bestOpp, true); err != nil {
			log.Warn("simulate notify failed", "error", err)
		}
	}

	duration := time.Since(start)

	if cfg.Pretty {
		fmt.Println(FormatBlockTable(BlockReport{
			Block:         block,
			EthMidUSD:     ethMid,
			GasUSD:        gasUSD,
			Evaluations:   evaluations,
			BlockBest:     blockBest,
			Opportunities: opportunities,
			MinProfitPct:  cfg.MinProfitPct,
			Duration:      duration,
			Simulated:     true,
		}))
	}

	log.Info("simulate mode finished",
		"opportunities", opportunities,
		"duration_ms", duration.Milliseconds(),
	)

	if opportunities == 0 {
		return fmt.Errorf("simulate expected at least one opportunity")
	}

	return nil
}

func simulatedBlock() ethereum.Block {
	return ethereum.Block{
		Number:    18234567,
		Hash:      "0xsimulated",
		Timestamp: time.Date(2024, 1, 15, 14, 23, 45, 0, time.UTC),
	}
}

func simulatedInput(
	block ethereum.Block,
	size decimal.Decimal,
	gasUSD decimal.Decimal,
	takerFee decimal.Decimal,
	minProfit decimal.Decimal,
) AnalysisInput {
	return AnalysisInput{
		BlockNumber:     block.Number,
		Timestamp:       block.Timestamp,
		TradeSizeETH:    size,
		CEXBuyUSD:       decimal.RequireFromString("2245.30"),
		CEXSellUSD:      decimal.RequireFromString("2240.00"),
		DEXSellUSD:      decimal.RequireFromString("2267.80"),
		DEXBuyUSD:       decimal.RequireFromString("2270.00"),
		GasCostUSD:      gasUSD,
		BinanceTakerFee: takerFee,
		MinProfitPct:    minProfit,
	}
}

func logSizeAnalysisFromEval(log *slog.Logger, blockNumber uint64, eval SizeEvaluation) {
	if log == nil {
		return
	}
	in := eval.Input
	log.Info("size analyzed [simulated]",
		"block", blockNumber,
		"size_eth", in.TradeSizeETH.StringFixed(1),
		"cex_buy_usd", in.CEXBuyUSD.StringFixed(2),
		"cex_sell_usd", in.CEXSellUSD.StringFixed(2),
		"dex_sell_usd", in.DEXSellUSD.StringFixed(2),
		"dex_buy_usd", in.DEXBuyUSD.StringFixed(2),
		"cex_to_dex_profit_usd", profitUSD(eval.CEXToDEX),
		"cex_to_dex_profit_pct", profitPct(eval.CEXToDEX),
		"dex_to_cex_profit_usd", profitUSD(eval.DEXToCEX),
		"dex_to_cex_profit_pct", profitPct(eval.DEXToCEX),
		"best_direction", shortDirection(eval.BestEffort),
		"best_profit_pct", formatProfitPct(eval.BestEffort),
		"min_profit_pct", in.MinProfitPct.StringFixed(2),
		"opportunity", eval.Opportunity != nil,
	)
}
