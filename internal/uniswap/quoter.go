package uniswap

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	appcommon "github.com/aramik/ced-dex-bot/internal/common"
	"github.com/aramik/ced-dex-bot/internal/config"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

// Quoter simulates Uniswap V3 swaps via QuoterV2 eth_call.
type Quoter interface {
	QuoteSellETH(ctx context.Context, ethAmount *big.Int, blockNumber *big.Int) (*Quote, error)
	QuoteBuyETH(ctx context.Context, ethAmount *big.Int, blockNumber *big.Int) (*Quote, error)
}

// QuoterV2 calls the on-chain QuoterV2 contract.
type QuoterV2 struct {
	client     *Client
	quoterABI  abi.ABI
	quoterAddr common.Address
	weth       common.Address
	usdc       common.Address
	fee        *big.Int
}

// NewQuoterV2 creates a QuoterV2 backed by HTTP RPC.
func NewQuoterV2(client *Client, cfg config.UniswapConfig) (*QuoterV2, error) {
	parsed, err := abi.JSON(strings.NewReader(quoterV2ABI))
	if err != nil {
		return nil, fmt.Errorf("parse quoter abi: %w", err)
	}

	return &QuoterV2{
		client:     client,
		quoterABI:  parsed,
		quoterAddr: common.HexToAddress(cfg.QuoterAddress),
		weth:       common.HexToAddress(cfg.WETHAddress),
		usdc:       common.HexToAddress(cfg.USDCAddress),
		fee:        big.NewInt(int64(cfg.Fee)),
	}, nil
}

// QuoteSellETH simulates selling ethAmount of WETH for USDC (exact input).
func (q *QuoterV2) QuoteSellETH(ctx context.Context, ethAmount *big.Int, blockNumber *big.Int) (*Quote, error) {
	if ethAmount == nil || ethAmount.Sign() <= 0 {
		return nil, fmt.Errorf("eth amount must be positive")
	}

	data, err := q.quoterABI.Pack(
		"quoteExactInputSingle",
		q.weth,
		q.usdc,
		q.fee,
		ethAmount,
		big.NewInt(0),
	)
	if err != nil {
		return nil, fmt.Errorf("pack quoteExactInputSingle: %w", err)
	}

	amountOut, blockNum, err := q.callUint256(ctx, data, blockNumber, "quoteExactInputSingle")
	if err != nil {
		return nil, err
	}

	return &Quote{
		Direction:      WETHToUSDC,
		AmountIn:       new(big.Int).Set(ethAmount),
		AmountOut:      amountOut,
		EffectivePrice: effectiveSellPrice(ethAmount, amountOut),
		BlockNumber:    blockNum,
	}, nil
}

// QuoteBuyETH simulates buying ethAmount of WETH with USDC (exact output).
func (q *QuoterV2) QuoteBuyETH(ctx context.Context, ethAmount *big.Int, blockNumber *big.Int) (*Quote, error) {
	if ethAmount == nil || ethAmount.Sign() <= 0 {
		return nil, fmt.Errorf("eth amount must be positive")
	}

	data, err := q.quoterABI.Pack(
		"quoteExactOutputSingle",
		q.usdc,
		q.weth,
		q.fee,
		ethAmount,
		big.NewInt(0),
	)
	if err != nil {
		return nil, fmt.Errorf("pack quoteExactOutputSingle: %w", err)
	}

	amountIn, blockNum, err := q.callUint256(ctx, data, blockNumber, "quoteExactOutputSingle")
	if err != nil {
		return nil, err
	}

	return &Quote{
		Direction:      USDCToWETH,
		AmountIn:       amountIn,
		AmountOut:      new(big.Int).Set(ethAmount),
		EffectivePrice: effectiveBuyPrice(amountIn, ethAmount),
		BlockNumber:    blockNum,
	}, nil
}

func (q *QuoterV2) callUint256(ctx context.Context, data []byte, blockNumber *big.Int, method string) (*big.Int, uint64, error) {
	msg := ethereum.CallMsg{
		To:   &q.quoterAddr,
		Data: data,
	}

	out, err := q.client.RPC().CallContract(ctx, msg, blockNumber)
	if err != nil {
		if revertData, ok := revertReturnData(err); ok && len(revertData) > 0 {
			out = revertData
		} else {
			return nil, 0, fmt.Errorf("%w: %v", appcommon.ErrQuoteFailed, err)
		}
	}

	results, err := q.quoterABI.Unpack(method, out)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: unpack %s: %v", appcommon.ErrQuoteFailed, method, err)
	}
	if len(results) == 0 {
		return nil, 0, appcommon.ErrQuoteFailed
	}

	amount, ok := results[0].(*big.Int)
	if !ok {
		return nil, 0, appcommon.ErrQuoteFailed
	}

	blockNum, err := q.resolveBlockNumber(ctx, blockNumber)
	if err != nil {
		return nil, 0, err
	}

	return amount, blockNum, nil
}

func (q *QuoterV2) resolveBlockNumber(ctx context.Context, blockNumber *big.Int) (uint64, error) {
	if blockNumber != nil {
		return blockNumber.Uint64(), nil
	}
	return q.client.BlockNumber(ctx)
}

// effectiveSellPrice returns USDC per ETH when selling WETH (exact input).
func effectiveSellPrice(amountInWei, amountOutUSDC *big.Int) decimal.Decimal {
	eth := rawToDecimal(amountInWei, appcommon.ETHDecimals)
	usdc := rawToDecimal(amountOutUSDC, appcommon.USDCDecimals)
	if eth.IsZero() {
		return decimal.Zero
	}
	return usdc.Div(eth)
}

// effectiveBuyPrice returns USDC per ETH when buying WETH (exact output).
func effectiveBuyPrice(amountInUSDC, amountOutWei *big.Int) decimal.Decimal {
	eth := rawToDecimal(amountOutWei, appcommon.ETHDecimals)
	usdc := rawToDecimal(amountInUSDC, appcommon.USDCDecimals)
	if eth.IsZero() {
		return decimal.Zero
	}
	return usdc.Div(eth)
}

func rawToDecimal(value *big.Int, decimals int) decimal.Decimal {
	if value == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(value, -int32(decimals))
}

// ETHToWei converts a human-readable ETH amount to wei.
func ETHToWei(eth decimal.Decimal) *big.Int {
	scaled := eth.Shift(int32(appcommon.ETHDecimals))
	return scaled.BigInt()
}

var _ Quoter = (*QuoterV2)(nil)
