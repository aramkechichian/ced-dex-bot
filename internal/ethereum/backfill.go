package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// BlockFetcher loads block headers by number (HTTP JSON-RPC backfill).
type BlockFetcher interface {
	FetchBlock(ctx context.Context, number uint64) (Block, error)
}

// HTTPBlockFetcher uses eth_getBlockByNumber via go-ethereum.
type HTTPBlockFetcher struct {
	rpc *ethclient.Client
}

// NewHTTPBlockFetcher creates a block fetcher backed by HTTP RPC.
func NewHTTPBlockFetcher(rpc *ethclient.Client) *HTTPBlockFetcher {
	return &HTTPBlockFetcher{rpc: rpc}
}

// FetchBlock returns the block header at number.
func (f *HTTPBlockFetcher) FetchBlock(ctx context.Context, number uint64) (Block, error) {
	if f == nil || f.rpc == nil {
		return Block{}, fmt.Errorf("block fetcher not configured")
	}

	header, err := f.rpc.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return Block{}, fmt.Errorf("header by number %d: %w", number, err)
	}
	if header == nil {
		return Block{}, fmt.Errorf("header by number %d: not found", number)
	}

	return Block{
		Number:    header.Number.Uint64(),
		Hash:      header.Hash().Hex(),
		Timestamp: time.Unix(int64(header.Time), 0).UTC(),
	}, nil
}

var _ BlockFetcher = (*HTTPBlockFetcher)(nil)
