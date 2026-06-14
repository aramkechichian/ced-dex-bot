package arbitrage

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/aramik/ced-dex-bot/internal/binance"
	"github.com/aramik/ced-dex-bot/internal/ethereum"
	"github.com/aramik/ced-dex-bot/internal/uniswap"
	"github.com/shopspring/decimal"
)

// ServiceConfig holds runtime parameters for the arbitrage orchestrator.
type ServiceConfig struct {
	Symbol          string
	TradeSizesETH   []decimal.Decimal
	BinanceTakerFee decimal.Decimal
	MinProfitPct    decimal.Decimal
	MaxBlocks       int // 0 = run until context cancelled
}

// Service wires block events to CEX/DEX pricing and the arbitrage detector.
type Service struct {
	cfg        ServiceConfig
	blocks     ethereum.BlockSubscriber
	exchange   binance.Exchange
	pricing    *binance.Service
	quoter     uniswap.Quoter
	detector   *Detector
	gas        GasCostEstimator
	log        *slog.Logger
	mu         sync.Mutex
	lastBlock  uint64
	processed  int
}

// NewService creates the arbitrage orchestrator.
func NewService(
	cfg ServiceConfig,
	blocks ethereum.BlockSubscriber,
	exchange binance.Exchange,
	pricing *binance.Service,
	quoter uniswap.Quoter,
	detector *Detector,
	gas GasCostEstimator,
	log *slog.Logger,
) *Service {
	return &Service{
		cfg:      cfg,
		blocks:   blocks,
		exchange: exchange,
		pricing:  pricing,
		quoter:   quoter,
		detector: detector,
		gas:      gas,
		log:      log,
	}
}

// Run subscribes to blocks and evaluates arbitrage until ctx is cancelled or MaxBlocks is reached.
func (s *Service) Run(ctx context.Context) error {
	if s.blocks == nil {
		return fmt.Errorf("block subscriber is required")
	}

	blocks, err := s.blocks.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("subscribe blocks: %w", err)
	}

	s.log.Info("arbitrage service started",
		"symbol", s.cfg.Symbol,
		"trade_sizes", tradeSizesString(s.cfg.TradeSizesETH),
		"min_profit_pct", s.cfg.MinProfitPct,
	)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("arbitrage service stopping", "reason", ctx.Err())
			return nil

		case block, ok := <-blocks:
			if !ok {
				s.log.Info("block stream closed")
				return nil
			}

			if !s.shouldProcess(block.Number) {
				continue
			}

			if err := s.processBlock(ctx, block); err != nil {
				s.log.Warn("block processing failed",
					"block", block.Number,
					"error", err,
				)
			}

			if s.reachedMaxBlocks() {
				s.log.Info("max blocks processed, stopping", "count", s.processed)
				return nil
			}
		}
	}
}

func (s *Service) shouldProcess(blockNumber uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if blockNumber <= s.lastBlock {
		return false
	}
	s.lastBlock = blockNumber
	return true
}

func (s *Service) reachedMaxBlocks() bool {
	if s.cfg.MaxBlocks <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processed >= s.cfg.MaxBlocks
}

func (s *Service) processBlock(ctx context.Context, block ethereum.Block) error {
	start := time.Now()

	book, err := s.exchange.GetOrderbook(ctx, s.cfg.Symbol)
	if err != nil {
		return fmt.Errorf("fetch orderbook: %w", err)
	}

	ethMid, err := midPrice(book)
	if err != nil {
		return fmt.Errorf("eth reference price: %w", err)
	}

	gasUSD, err := s.gas.SwapCostUSD(ctx, ethMid)
	if err != nil {
		return fmt.Errorf("estimate gas: %w", err)
	}

	blockNum := new(big.Int).SetUint64(block.Number)
	opportunities := 0
	sizesChecked := 0

	for _, size := range s.cfg.TradeSizesETH {
		opp, err := s.analyzeSize(ctx, block, book, blockNum, size, gasUSD)
		if err != nil {
			s.log.Warn("trade size skipped",
				"block", block.Number,
				"size_eth", size,
				"error", err,
			)
			continue
		}
		sizesChecked++

		if opp != nil {
			opportunities++
			fmt.Println(FormatOpportunity(opp))
		}
	}

	s.mu.Lock()
	s.processed++
	count := s.processed
	s.mu.Unlock()

	s.log.Info("block analyzed",
		"block", block.Number,
		"hash", block.Hash,
		"sizes_checked", sizesChecked,
		"opportunities", opportunities,
		"gas_usd", gasUSD.StringFixed(2),
		"eth_mid_usd", ethMid.StringFixed(2),
		"duration_ms", time.Since(start).Milliseconds(),
		"processed", count,
	)

	return nil
}

func (s *Service) analyzeSize(
	ctx context.Context,
	block ethereum.Block,
	book *binance.Orderbook,
	blockNum *big.Int,
	size decimal.Decimal,
	gasUSD decimal.Decimal,
) (*Opportunity, error) {
	cexPrices, err := s.pricing.EffectivePrices(book, size)
	if err != nil {
		return nil, fmt.Errorf("cex prices: %w", err)
	}

	ethWei := uniswap.ETHToWei(size)

	sellQuote, err := s.quoter.QuoteSellETH(ctx, ethWei, blockNum)
	if err != nil {
		return nil, fmt.Errorf("dex sell quote: %w", err)
	}

	buyQuote, err := s.quoter.QuoteBuyETH(ctx, ethWei, blockNum)
	if err != nil {
		return nil, fmt.Errorf("dex buy quote: %w", err)
	}

	input := AnalysisInput{
		BlockNumber:     block.Number,
		Timestamp:       block.Timestamp,
		TradeSizeETH:    size,
		CEXBuyUSD:       cexPrices.EffectiveBuyUSD,
		CEXSellUSD:      cexPrices.EffectiveSellUSD,
		DEXSellUSD:      sellQuote.EffectivePrice,
		DEXBuyUSD:       buyQuote.EffectivePrice,
		GasCostUSD:      gasUSD,
		BinanceTakerFee: s.cfg.BinanceTakerFee,
		MinProfitPct:    s.cfg.MinProfitPct,
	}

	return s.detector.Analyze(input), nil
}

func midPrice(book *binance.Orderbook) (decimal.Decimal, error) {
	bid, ask, ok := binance.BestBidAsk(book)
	if !ok {
		return decimal.Zero, fmt.Errorf("empty orderbook")
	}
	return bid.Price.Add(ask.Price).Div(decimal.NewFromInt(2)), nil
}

func tradeSizesString(sizes []decimal.Decimal) string {
	if len(sizes) == 0 {
		return "[]"
	}
	out := sizes[0].String()
	for i := 1; i < len(sizes); i++ {
		out += "," + sizes[i].String()
	}
	return out
}
