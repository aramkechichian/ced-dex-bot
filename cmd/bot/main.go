package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aramik/ced-dex-bot/internal/common"
	"github.com/aramik/ced-dex-bot/internal/config"
	"github.com/aramik/ced-dex-bot/internal/logger"
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

	log.Info("ced-dex-bot starting",
		"phase", "1-constants",
		"binance_symbol", cfg.Binance.Symbol,
		"trade_sizes_eth", cfg.Arbitrage.TradeSizesETH,
		"min_profit_pct", cfg.Arbitrage.MinProfitPct,
		"uniswap_pool", cfg.Uniswap.PoolAddress,
		"weth", common.WETHAddress,
		"eth_decimals", common.ETHDecimals,
		"usdc_decimals", common.USDCDecimals,
	)

	fmt.Println()
	fmt.Println("=== Configuration loaded successfully ===")
	fmt.Printf("Binance symbol:     %s\n", cfg.Binance.Symbol)
	fmt.Printf("Trade sizes (ETH):  %v\n", cfg.Arbitrage.TradeSizesETH)
	fmt.Printf("Min profit %%:       %.2f\n", cfg.Arbitrage.MinProfitPct)
	fmt.Printf("Uniswap pool:       %s\n", cfg.Uniswap.PoolAddress)
	fmt.Printf("Ethereum HTTP URL:  %s\n", maskURL(cfg.Ethereum.HTTPURL))
	fmt.Printf("Ethereum WS URL:    %s\n", maskURL(cfg.Ethereum.WSURL))
	fmt.Println()
	fmt.Println("Phase 1 complete — constants and errors are ready.")
}

func maskURL(url string) string {
	if idx := strings.LastIndex(url, "/v3/"); idx != -1 {
		return url[:idx+4] + "***"
	}
	return url
}
