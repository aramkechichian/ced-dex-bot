package uniswap

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps the Ethereum JSON-RPC HTTP connection.
type Client struct {
	rpc *ethclient.Client
}

// Dial connects to an Ethereum HTTP RPC endpoint (Infura, Alchemy, etc.).
func Dial(ctx context.Context, httpURL string) (*Client, error) {
	rpc, err := ethclient.DialContext(ctx, httpURL)
	if err != nil {
		return nil, fmt.Errorf("dial ethereum rpc: %w", err)
	}
	return &Client{rpc: rpc}, nil
}

// RPC returns the underlying ethclient for contract calls.
func (c *Client) RPC() *ethclient.Client {
	return c.rpc
}

// Close closes the RPC connection.
func (c *Client) Close() {
	if c.rpc != nil {
		c.rpc.Close()
	}
}

// BlockNumber returns the latest block number.
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.rpc.BlockNumber(ctx)
}
