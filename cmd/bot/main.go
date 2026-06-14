package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aramik/ced-dex-bot/internal/config"
	"github.com/aramik/ced-dex-bot/internal/logger"
	"github.com/aramik/ced-dex-bot/internal/uniswap"
	"github.com/shopspring/decimal"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	envPath := flag.String("env", ".env", "path to .env file (optional)")
	flag.Parse()

	log := logger.New("info")

	if err := config.LoadEnvFile(*envPath); err != nil {
		log.Error("failed to load env file", "path", *envPath, "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("failed to load config", "path", *configPath, "error", err)
		os.Exit(1)
	}

	log = logger.New(cfg.Logging.Level)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rpcClient, err := uniswap.Dial(ctx, cfg.Ethereum.HTTPURL)
	if err != nil {
		log.Error("failed to connect ethereum rpc", "error", err)
		os.Exit(1)
	}
	defer rpcClient.Close()

	quoter, err := uniswap.NewQuoterV2(rpcClient, cfg.Uniswap)
	if err != nil {
		log.Error("failed to create quoter", "error", err)
		os.Exit(1)
	}

	blockNum, err := rpcClient.BlockNumber(ctx)
	if err != nil {
		log.Error("failed to get block number", "error", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== Uniswap V3 QuoterV2 (mainnet) ===")
	fmt.Printf("Block:        #%d\n", blockNum)
	fmt.Printf("Pool fee:     0.3%% (tier %d)\n", cfg.Uniswap.Fee)
	fmt.Println()

	for _, size := range cfg.Arbitrage.TradeSizesETH {
		ethAmount := uniswap.ETHToWei(decimal.NewFromFloat(size))

		sellQuote, err := quoter.QuoteSellETH(ctx, ethAmount, nil)
		if err != nil {
			log.Error("sell quote failed", "size_eth", size, "error", err)
			continue
		}

		buyQuote, err := quoter.QuoteBuyETH(ctx, ethAmount, nil)
		if err != nil {
			log.Error("buy quote failed", "size_eth", size, "error", err)
			continue
		}

		usdcOut := decimal.NewFromBigInt(sellQuote.AmountOut, -6)
		usdcIn := decimal.NewFromBigInt(buyQuote.AmountIn, -6)

		fmt.Printf("%g ETH:\n", size)
		fmt.Printf("  Sell on Uniswap (WETH→USDC): $%s/ETH → receive ~$%s USDC\n",
			sellQuote.EffectivePrice.StringFixed(2),
			usdcOut.StringFixed(2),
		)
		fmt.Printf("  Buy on Uniswap  (USDC→WETH): $%s/ETH → spend   ~$%s USDC\n",
			buyQuote.EffectivePrice.StringFixed(2),
			usdcIn.StringFixed(2),
		)
		fmt.Println()

		log.Info("uniswap quote",
			"size_eth", size,
			"sell_usd_per_eth", sellQuote.EffectivePrice.StringFixed(2),
			"buy_usd_per_eth", buyQuote.EffectivePrice.StringFixed(2),
			"block", sellQuote.BlockNumber,
		)
	}

	fmt.Println("Phase 6 complete — Uniswap quoter is working.")
}
