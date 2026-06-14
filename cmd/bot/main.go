package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aramik/ced-dex-bot/internal/arbitrage"
	"github.com/aramik/ced-dex-bot/internal/binance"
	"github.com/aramik/ced-dex-bot/internal/cache"
	"github.com/aramik/ced-dex-bot/internal/config"
	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/aramik/ced-dex-bot/internal/logger"
	"github.com/aramik/ced-dex-bot/internal/resilience"
	"github.com/aramik/ced-dex-bot/internal/uniswap"
	"github.com/shopspring/decimal"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	envPath := flag.String("env", ".env", "path to .env file (optional)")
	maxBlocks := flag.Int("blocks", 0, "stop after N blocks (0 = run until interrupted)")
	flag.Parse()

	if err := config.LoadEnvFile(*envPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load env file %s: %v\n", *envPath, err)
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Logging.Level)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	uniClient, err := uniswap.Dial(ctx, cfg.Ethereum.HTTPURL)
	if err != nil {
		log.Error("failed to connect ethereum rpc", "error", err)
		os.Exit(1)
	}
	defer uniClient.Close()

	baseQuoter, err := uniswap.NewQuoterV2(uniClient, cfg.Uniswap)
	if err != nil {
		log.Error("failed to create uniswap quoter", "error", err)
		os.Exit(1)
	}

	memCache := cache.NewMemoryCache()
	gasEst := arbitrage.NewCachedGasEstimator(
		arbitrage.NewGasEstimator(uniClient.RPC()),
		memCache,
		time.Duration(cfg.Resilience.GasCacheTTLSecondsOrDefault())*time.Second,
		log,
	)

	retryCfg := resilience.RetryConfig{
		MaxAttempts: cfg.Resilience.RPCMaxRetriesOrDefault(),
		BaseDelay:   time.Duration(cfg.Resilience.RPCRetryBaseMSOrDefault()) * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}

	binanceClient := resilience.NewRateLimitedExchange(
		binance.NewClient(cfg.Binance.BaseURL, cfg.Binance.OrderbookLimit),
		cfg.Resilience.BinanceRPSOrDefault(),
		cfg.Resilience.BinanceBurstOrDefault(),
	)

	quoter := resilience.NewRetryingQuoter(baseQuoter, retryCfg, log)
	blockSub := ethereum.NewWebSocketSubscriber(cfg.Ethereum.WSURL, log)

	svc := arbitrage.NewService(
		arbitrage.ServiceConfig{
			Symbol:          cfg.Binance.Symbol,
			TradeSizesETH:   tradeSizesFromConfig(cfg.Arbitrage.TradeSizesETH),
			BinanceTakerFee: decimal.NewFromFloat(cfg.Arbitrage.BinanceTakerFee),
			MinProfitPct:    decimal.NewFromFloat(cfg.Arbitrage.MinProfitPct),
			MaxBlocks:       *maxBlocks,
		},
		blockSub,
		binanceClient,
		binance.NewService(),
		quoter,
		arbitrage.NewDetector(),
		gasEst,
		log,
	)

	log.Info("ced-dex-bot starting",
		"symbol", cfg.Binance.Symbol,
		"max_blocks", *maxBlocks,
		"gas_cache_ttl_s", cfg.Resilience.GasCacheTTLSecondsOrDefault(),
		"binance_rps", cfg.Resilience.BinanceRPSOrDefault(),
		"rpc_max_retries", cfg.Resilience.RPCMaxRetriesOrDefault(),
	)

	if err := svc.Run(ctx); err != nil {
		log.Error("service exited with error", "error", err)
		os.Exit(1)
	}

	log.Info("ced-dex-bot stopped")
}

func tradeSizesFromConfig(sizes []float64) []decimal.Decimal {
	out := make([]decimal.Decimal, len(sizes))
	for i, s := range sizes {
		out[i] = decimal.NewFromFloat(s)
	}
	return out
}
