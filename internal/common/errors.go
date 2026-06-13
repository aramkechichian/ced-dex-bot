package common

import "errors"

// Sentinel errors used across packages.
// Callers can branch with errors.Is(err, common.ErrRateLimited) to decide
// whether to retry, reconnect, or fail fast.
var (
	// ErrInsufficientLiquidity means the orderbook cannot fill the requested trade size.
	ErrInsufficientLiquidity = errors.New("insufficient liquidity in orderbook")

	// ErrInvalidOrderbook means the CEX response was empty or malformed.
	ErrInvalidOrderbook = errors.New("invalid orderbook data")

	// ErrConnectionLost means the Ethereum WebSocket dropped.
	ErrConnectionLost = errors.New("websocket connection lost")

	// ErrRateLimited means an external API rejected the request due to rate limits.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrCircuitOpen means the circuit breaker blocked the call to a failing service.
	ErrCircuitOpen = errors.New("circuit breaker open")

	// ErrQuoteFailed means the Uniswap QuoterV2 call returned no usable result.
	ErrQuoteFailed = errors.New("uniswap quote failed")

	// ErrBlockGap means blocks were missed and need backfill after reconnect.
	ErrBlockGap = errors.New("block gap detected")
)
