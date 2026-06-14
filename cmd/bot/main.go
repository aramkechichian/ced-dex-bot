package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aramik/ced-dex-bot/internal/arbitrage"
	"github.com/aramik/ced-dex-bot/internal/binance"
	"github.com/aramik/ced-dex-bot/internal/cache"
	"github.com/aramik/ced-dex-bot/internal/config"
	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/aramik/ced-dex-bot/internal/logger"
	"github.com/aramik/ced-dex-bot/internal/notify"
	"github.com/aramik/ced-dex-bot/internal/resilience"
	"github.com/aramik/ced-dex-bot/internal/uniswap"
	"github.com/shopspring/decimal"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	envPath := flag.String("env", ".env", "path to .env file (optional)")
	maxBlocks := flag.Int("blocks", 0, "stop after N blocks (0 = run until interrupted)")
	pretty := flag.Bool("pretty", false, "print human-readable table per block")
	simulate := flag.Bool("simulate", false, "offline demo with challenge PDF prices (no network)")
	notifyFlag := flag.Bool("notify", false, "POST opportunity to ACTION_WEBHOOK_URL when set")
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

	svcCfg := arbitrage.ServiceConfig{
		Symbol:          cfg.Binance.Symbol,
		TradeSizesETH:   tradeSizesFromConfig(cfg.Arbitrage.TradeSizesETH),
		BinanceTakerFee: decimal.NewFromFloat(cfg.Arbitrage.BinanceTakerFee),
		MinProfitPct:    decimal.NewFromFloat(cfg.Arbitrage.MinProfitPct),
		MaxBlocks:       *maxBlocks,
		Pretty:          *pretty,
	}

	notifier := buildActionNotifier(os.Getenv("ACTION_WEBHOOK_URL"))
	webhookSet := strings.TrimSpace(os.Getenv("ACTION_WEBHOOK_URL")) != ""
	notifyEnabled := *notifyFlag || webhookSet
	if *notifyFlag && !webhookSet {
		log.Warn("notify requested but ACTION_WEBHOOK_URL is not set; action webhook will be skipped")
	}
	emitter := arbitrage.NewOpportunityEmitter(notifier, notifyEnabled && webhookSet)

	if *simulate {
		log.Info("ced-dex-bot simulate mode", "notify", notifyEnabled && webhookSet)
		if err := arbitrage.RunSimulate(ctx, svcCfg, arbitrage.NewDetector(), emitter, log); err != nil {
			log.Error("simulate failed", "error", err)
			os.Exit(1)
		}
		log.Info("ced-dex-bot stopped")
		return
	}

	if err := cfg.ValidateLive(); err != nil {
		log.Error("live mode config invalid", "error", err)
		os.Exit(1)
	}

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
	blockSub.ConfigureBackfill(
		ethereum.NewHTTPBlockFetcher(uniClient.RPC()),
		cfg.Ethereum.MaxBackfillBlocksOrDefault(),
	)

	svc := arbitrage.NewService(
		svcCfg,
		blockSub,
		binanceClient,
		binance.NewService(),
		quoter,
		arbitrage.NewDetector(),
		gasEst,
		emitter,
		log,
	)

	log.Info("ced-dex-bot starting",
		"symbol", cfg.Binance.Symbol,
		"max_blocks", *maxBlocks,
		"pretty", *pretty,
		"notify", notifyEnabled && webhookSet,
		"gas_cache_ttl_s", cfg.Resilience.GasCacheTTLSecondsOrDefault(),
		"binance_rps", cfg.Resilience.BinanceRPSOrDefault(),
		"rpc_max_retries", cfg.Resilience.RPCMaxRetriesOrDefault(),
		"max_backfill_blocks", cfg.Ethereum.MaxBackfillBlocksOrDefault(),
	)

	if err := svc.Run(ctx); err != nil {
		log.Error("service exited with error", "error", err)
		os.Exit(1)
	}

	log.Info("ced-dex-bot stopped")
}

func buildActionNotifier(webhookURL string) notify.Notifier {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return notify.NoOpNotifier{}
	}
	return notify.NewActionWebhook(webhookURL)
}

func tradeSizesFromConfig(sizes []float64) []decimal.Decimal {
	out := make([]decimal.Decimal, len(sizes))
	for i, s := range sizes {
		out[i] = decimal.NewFromFloat(s)
	}
	return out
}
