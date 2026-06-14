package arbitrage

import (
	"context"
	"fmt"
	"math/big"

	appcommon "github.com/aramik/ced-dex-bot/internal/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// GasCostEstimator converts on-chain gas price to USD.
type GasCostEstimator interface {
	SwapCostUSD(ctx context.Context, ethPriceUSD decimal.Decimal) (decimal.Decimal, error)
}

// GasEstimator fetches suggested gas price via JSON-RPC.
type GasEstimator struct {
	rpc *ethclient.Client
}

// NewGasEstimator creates a gas cost estimator backed by JSON-RPC.
func NewGasEstimator(rpc *ethclient.Client) *GasEstimator {
	return &GasEstimator{rpc: rpc}
}

// SwapCostUSD estimates the USD cost of a Uniswap swap at the current suggested gas price.
// ethPriceUSD should be a recent ETH/USDC reference (e.g. CEX mid price).
func (g *GasEstimator) SwapCostUSD(ctx context.Context, ethPriceUSD decimal.Decimal) (decimal.Decimal, error) {
	if g == nil || g.rpc == nil {
		return decimal.Zero, fmt.Errorf("gas estimator not configured")
	}
	if ethPriceUSD.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("eth price must be positive")
	}

	gasPrice, err := g.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("suggest gas price: %w", err)
	}

	return swapCostFromGasPrice(gasPrice, ethPriceUSD), nil
}

func swapCostFromGasPrice(gasPrice *big.Int, ethPriceUSD decimal.Decimal) decimal.Decimal {
	gasUnits := new(big.Int).SetUint64(appcommon.EstimatedSwapGasUnits)
	gasCostWei := new(big.Int).Mul(gasPrice, gasUnits)
	ethCost := weiToETH(gasCostWei)
	return ethCost.Mul(ethPriceUSD)
}

var _ GasCostEstimator = (*GasEstimator)(nil)

func weiToETH(wei *big.Int) decimal.Decimal {
	if wei == nil {
		return decimal.Zero
	}
	scale := decimal.NewFromInt(1).Shift(int32(appcommon.ETHDecimals))
	return decimal.NewFromBigInt(wei, 0).Div(scale)
}
