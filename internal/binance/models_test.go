package binance

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestOrderbookLevels(t *testing.T) {
	book := Orderbook{
		Symbol: "ETHUSDC",
		Bids: []Level{
			{Price: decimal.NewFromFloat(2245.30), Quantity: decimal.NewFromFloat(1.5)},
		},
		Asks: []Level{
			{Price: decimal.NewFromFloat(2245.50), Quantity: decimal.NewFromFloat(1.2)},
		},
	}

	if !book.Bids[0].Price.LessThan(book.Asks[0].Price) {
		t.Fatal("expected bid below ask in a normal book")
	}
}
