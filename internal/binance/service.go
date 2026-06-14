package binance

import (
	"fmt"

	"github.com/aramik/ced-dex-bot/internal/common"
	"github.com/shopspring/decimal"
)

// Service computes effective execution prices from orderbook snapshots.
type Service struct{}

// NewService creates a binance pricing service.
func NewService() *Service {
	return &Service{}
}

// EffectiveBuyPrice returns the average USDC per ETH when buying ethAmount on the CEX.
// Walks asks from lowest price upward.
func (s *Service) EffectiveBuyPrice(book *Orderbook, ethAmount decimal.Decimal) (decimal.Decimal, error) {
	totalUSDC, err := walkLevels(book.Asks, ethAmount)
	if err != nil {
		return decimal.Zero, fmt.Errorf("effective buy price: %w", err)
	}
	return totalUSDC.Div(ethAmount), nil
}

// EffectiveSellPrice returns the average USDC per ETH when selling ethAmount on the CEX.
// Walks bids from highest price downward.
func (s *Service) EffectiveSellPrice(book *Orderbook, ethAmount decimal.Decimal) (decimal.Decimal, error) {
	totalUSDC, err := walkLevels(book.Bids, ethAmount)
	if err != nil {
		return decimal.Zero, fmt.Errorf("effective sell price: %w", err)
	}
	return totalUSDC.Div(ethAmount), nil
}

// EffectivePrices returns buy and sell effective prices for a trade size.
func (s *Service) EffectivePrices(book *Orderbook, ethAmount decimal.Decimal) (*SidePrices, error) {
	if book == nil {
		return nil, common.ErrInvalidOrderbook
	}
	if ethAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("trade size must be positive")
	}

	buy, err := s.EffectiveBuyPrice(book, ethAmount)
	if err != nil {
		return nil, err
	}
	sell, err := s.EffectiveSellPrice(book, ethAmount)
	if err != nil {
		return nil, err
	}

	return &SidePrices{
		TradeSizeETH:     ethAmount,
		EffectiveBuyUSD:  buy,
		EffectiveSellUSD: sell,
	}, nil
}

// walkLevels consumes ethAmount across orderbook levels and returns total USDC.
func walkLevels(levels []Level, ethAmount decimal.Decimal) (decimal.Decimal, error) {
	if len(levels) == 0 {
		return decimal.Zero, common.ErrInvalidOrderbook
	}

	remaining := ethAmount
	totalUSDC := decimal.Zero

	for _, level := range levels {
		if remaining.IsZero() {
			break
		}

		fill := decimal.Min(remaining, level.Quantity)
		if fill.IsZero() {
			continue
		}

		totalUSDC = totalUSDC.Add(fill.Mul(level.Price))
		remaining = remaining.Sub(fill)
	}

	if remaining.GreaterThan(decimal.Zero) {
		return decimal.Zero, common.ErrInsufficientLiquidity
	}

	return totalUSDC, nil
}
