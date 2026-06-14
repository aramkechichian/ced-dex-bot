package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aramik/ced-dex-bot/internal/binance"
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	binanceClient := binance.NewClient(cfg.Binance.BaseURL, cfg.Binance.OrderbookLimit)
	book, err := binanceClient.GetOrderbook(ctx, cfg.Binance.Symbol)
	if err != nil {
		log.Error("failed to fetch binance orderbook", "symbol", cfg.Binance.Symbol, "error", err)
		os.Exit(1)
	}

	bid, ask, _ := binance.BestBidAsk(book)
	spread, _ := binance.Spread(book)

	log.Info("ced-dex-bot starting",
		"phase", "3-binance-client",
		"symbol", book.Symbol,
		"levels_bids", len(book.Bids),
		"levels_asks", len(book.Asks),
		"best_bid", bid.Price.StringFixed(2),
		"best_ask", ask.Price.StringFixed(2),
		"spread", spread.StringFixed(2),
	)

	fmt.Println()
	fmt.Println("=== Binance orderbook fetched successfully ===")
	fmt.Printf("Symbol:       %s\n", book.Symbol)
	fmt.Printf("Best bid:     $%s (%s ETH)\n", bid.Price.StringFixed(2), bid.Quantity.StringFixed(4))
	fmt.Printf("Best ask:     $%s (%s ETH)\n", ask.Price.StringFixed(2), ask.Quantity.StringFixed(4))
	fmt.Printf("Spread:       $%s\n", spread.StringFixed(2))
	fmt.Printf("Bid levels:   %d\n", len(book.Bids))
	fmt.Printf("Ask levels:   %d\n", len(book.Asks))
	fmt.Printf("Last update:  %d\n", book.LastUpdateID)
	fmt.Printf("Fetched at:   %s\n", book.FetchedAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Phase 3 complete — Binance client is working.")
}
