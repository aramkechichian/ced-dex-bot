package uniswap

import (
	"math/big"

	"github.com/shopspring/decimal"
)

// SwapDirection indicates which tokens flow through the pool.
type SwapDirection int

const (
	// WETHToUSDC sells ETH on Uniswap (input WETH, output USDC).
	WETHToUSDC SwapDirection = iota
	// USDCToWETH buys ETH on Uniswap (input USDC, output WETH).
	USDCToWETH
)

func (d SwapDirection) String() string {
	switch d {
	case WETHToUSDC:
		return "WETH → USDC"
	case USDCToWETH:
		return "USDC → WETH"
	default:
		return "unknown"
	}
}

// Quote is the result of a QuoterV2 simulation at a specific block.
type Quote struct {
	Direction      SwapDirection
	AmountIn       *big.Int // raw on-chain units (wei or micro-USDC)
	AmountOut      *big.Int
	EffectivePrice decimal.Decimal // USDC per ETH, human-readable
	BlockNumber    uint64
}
