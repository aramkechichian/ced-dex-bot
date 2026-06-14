package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aramik/ced-dex-bot/internal/common"
	"github.com/shopspring/decimal"
)

const defaultTimeout = 10 * time.Second

// Client calls the Binance Spot REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	limit      int
}

// NewClient creates a Binance exchange client.
func NewClient(baseURL string, limit int) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		limit: limit,
	}
}

// depthResponse is the raw JSON from GET /api/v3/depth.
type depthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// GetOrderbook fetches a snapshot of the orderbook for symbol (e.g. ETHUSDC).
func (c *Client) GetOrderbook(ctx context.Context, symbol string) (*Orderbook, error) {
	endpoint, err := c.buildDepthURL(symbol)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance depth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("binance depth: %w", common.ErrRateLimited)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("binance depth status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw depthResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode binance depth: %w", err)
	}

	bids, err := parseLevels(raw.Bids)
	if err != nil {
		return nil, fmt.Errorf("parse bids: %w", err)
	}
	asks, err := parseLevels(raw.Asks)
	if err != nil {
		return nil, fmt.Errorf("parse asks: %w", err)
	}

	if len(bids) == 0 || len(asks) == 0 {
		return nil, common.ErrInvalidOrderbook
	}

	return &Orderbook{
		Symbol:       symbol,
		Bids:         bids,
		Asks:         asks,
		LastUpdateID: raw.LastUpdateID,
		FetchedAt:    time.Now().UTC(),
	}, nil
}

func (c *Client) buildDepthURL(symbol string) (string, error) {
	u, err := url.Parse(c.baseURL + "/api/v3/depth")
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	q := u.Query()
	q.Set("symbol", symbol)
	q.Set("limit", strconv.Itoa(c.limit))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func parseLevels(rows [][]string) ([]Level, error) {
	levels := make([]Level, 0, len(rows))
	for i, row := range rows {
		if len(row) != 2 {
			return nil, fmt.Errorf("row %d: expected [price, qty], got %d fields", i, len(row))
		}

		price, err := decimal.NewFromString(row[0])
		if err != nil {
			return nil, fmt.Errorf("row %d price: %w", i, err)
		}
		qty, err := decimal.NewFromString(row[1])
		if err != nil {
			return nil, fmt.Errorf("row %d qty: %w", i, err)
		}
		if price.IsNegative() || qty.IsNegative() {
			return nil, fmt.Errorf("row %d: negative price or quantity", i)
		}

		levels = append(levels, Level{Price: price, Quantity: qty})
	}
	return levels, nil
}

// BestBidAsk returns the top-of-book bid and ask.
func BestBidAsk(book *Orderbook) (bid, ask Level, ok bool) {
	if book == nil || len(book.Bids) == 0 || len(book.Asks) == 0 {
		return Level{}, Level{}, false
	}
	return book.Bids[0], book.Asks[0], true
}

// Spread returns ask minus bid (USDC per ETH).
func Spread(book *Orderbook) (decimal.Decimal, bool) {
	bid, ask, ok := BestBidAsk(book)
	if !ok {
		return decimal.Zero, false
	}
	return ask.Price.Sub(bid.Price), true
}
