package binance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aramik/ced-dex-bot/internal/common"
	"github.com/shopspring/decimal"
)

func TestClientGetOrderbook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/depth" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "ETHUSDC" {
			t.Fatalf("unexpected symbol: %s", r.URL.Query().Get("symbol"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"lastUpdateId": 123,
			"bids": [["2245.30", "1.5"], ["2245.20", "2.0"]],
			"asks": [["2245.50", "1.2"], ["2245.60", "0.8"]]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 100)
	book, err := client.GetOrderbook(context.Background(), "ETHUSDC")
	if err != nil {
		t.Fatalf("GetOrderbook: %v", err)
	}

	if book.LastUpdateID != 123 {
		t.Fatalf("lastUpdateId: %d", book.LastUpdateID)
	}
	if len(book.Bids) != 2 || len(book.Asks) != 2 {
		t.Fatalf("unexpected levels: bids=%d asks=%d", len(book.Bids), len(book.Asks))
	}

	bid, ask, ok := BestBidAsk(book)
	if !ok {
		t.Fatal("expected best bid/ask")
	}
	if !bid.Price.Equal(decimal.RequireFromString("2245.30")) {
		t.Fatalf("bid price: %s", bid.Price)
	}
	if !ask.Price.Equal(decimal.RequireFromString("2245.50")) {
		t.Fatalf("ask price: %s", ask.Price)
	}

	spread, ok := Spread(book)
	if !ok || !spread.Equal(decimal.RequireFromString("0.20")) {
		t.Fatalf("spread: %s", spread)
	}
}

func TestClientGetOrderbookRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL, 100)
	_, err := client.GetOrderbook(context.Background(), "ETHUSDC")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, common.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
}

func TestClientGetOrderbookEmptyBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lastUpdateId": 1, "bids": [], "asks": [["1","1"]]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 100)
	_, err := client.GetOrderbook(context.Background(), "ETHUSDC")
	if err != common.ErrInvalidOrderbook {
		t.Fatalf("expected ErrInvalidOrderbook, got: %v", err)
	}
}
