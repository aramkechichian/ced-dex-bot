package binance

import "context"

// Exchange fetches CEX market data. Implementations: Client (Binance REST).
type Exchange interface {
	GetOrderbook(ctx context.Context, symbol string) (*Orderbook, error)
}
