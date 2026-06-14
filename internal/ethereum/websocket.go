package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aramik/ced-dex-bot/internal/common"
	"github.com/gorilla/websocket"
)

const defaultMaxBackfillBlocks = 10

// WebSocketSubscriber streams newHeads from an Ethereum JSON-RPC WebSocket endpoint.
type WebSocketSubscriber struct {
	wsURL             string
	logger            *slog.Logger
	mu                sync.Mutex
	lastBlock         uint64
	fetcher           BlockFetcher
	maxBackfillBlocks uint64
}

// NewWebSocketSubscriber creates a block streamer for eth_subscribe/newHeads.
func NewWebSocketSubscriber(wsURL string, logger *slog.Logger) *WebSocketSubscriber {
	return &WebSocketSubscriber{
		wsURL:             wsURL,
		logger:            logger,
		maxBackfillBlocks: defaultMaxBackfillBlocks,
	}
}

// ConfigureBackfill enables HTTP backfill when block gaps are detected after reconnect.
func (s *WebSocketSubscriber) ConfigureBackfill(fetcher BlockFetcher, maxBlocks uint64) {
	s.fetcher = fetcher
	if maxBlocks > 0 {
		s.maxBackfillBlocks = maxBlocks
	}
}

// Subscribe returns a channel of blocks. A background goroutine maintains the connection.
func (s *WebSocketSubscriber) Subscribe(ctx context.Context) (<-chan Block, error) {
	if s.wsURL == "" {
		return nil, fmt.Errorf("websocket url is required")
	}

	out := make(chan Block, 16)
	go s.run(ctx, out)
	return out, nil
}

func (s *WebSocketSubscriber) run(ctx context.Context, out chan<- Block) {
	defer close(out)

	attempt := 0
	for {
		if ctx.Err() != nil {
			return
		}

		err := s.consume(ctx, out)
		if ctx.Err() != nil {
			return
		}

		s.logger.Warn("ethereum websocket disconnected, reconnecting",
			"error", err,
			"attempt", attempt+1,
			"delay", reconnectDelay(attempt).String(),
		)

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay(attempt)):
		}
		attempt++
	}
}

func (s *WebSocketSubscriber) consume(ctx context.Context, out chan<- Block) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, s.wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", common.ErrConnectionLost)
	}
	defer conn.Close()

	s.logger.Info("ethereum websocket connected")

	if err := conn.WriteJSON(newSubscribeMessage()); err != nil {
		return fmt.Errorf("subscribe newHeads: %w", err)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return fmt.Errorf("set read deadline: %w", err)
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %w", common.ErrConnectionLost)
		}

		block, ok, err := parseBlockFromSubscription(data)
		if err != nil {
			s.logger.Warn("failed to parse block message", "error", err)
			continue
		}
		if !ok {
			var ack rpcMessage
			if err := json.Unmarshal(data, &ack); err == nil && ack.ID != nil {
				s.logger.Info("ethereum subscription confirmed")
			}
			continue
		}

		if err := s.emitBlock(ctx, out, block); err != nil {
			return err
		}
	}
}

func (s *WebSocketSubscriber) emitBlock(ctx context.Context, out chan<- Block, block Block) error {
	last := s.LastBlock()

	if last > 0 && block.Number > last+1 {
		if err := s.backfillGap(ctx, out, last, block.Number); err != nil {
			return err
		}
	}

	return s.sendBlock(ctx, out, block)
}

func (s *WebSocketSubscriber) backfillGap(ctx context.Context, out chan<- Block, last, current uint64) error {
	from, to, missed, truncated := s.backfillRange(last, current)

	if s.logger != nil {
		s.logger.Warn("block gap detected",
			"last", last,
			"current", current,
			"missed", missed,
			"backfill_from", from,
			"backfill_to", to,
			"truncated", truncated,
			"error", common.ErrBlockGap,
		)
	}

	if s.fetcher == nil {
		if s.logger != nil {
			s.logger.Warn("block backfill skipped: no HTTP fetcher configured")
		}
		return nil
	}

	if s.logger != nil {
		s.logger.Info("backfilling missed blocks via HTTP",
			"from", from,
			"to", to,
			"count", to-from+1,
		)
	}

	for n := from; n <= to; n++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		b, err := s.fetcher.FetchBlock(ctx, n)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("backfill block fetch failed",
					"block", n,
					"error", err,
				)
			}
			break
		}

		if err := s.sendBlock(ctx, out, b); err != nil {
			return err
		}
	}

	return nil
}

func (s *WebSocketSubscriber) backfillRange(last, current uint64) (from, to, missed uint64, truncated bool) {
	from = last + 1
	to = current - 1
	if current <= last+1 {
		return from, to, 0, false
	}
	missed = to - from + 1
	if s.maxBackfillBlocks > 0 && missed > s.maxBackfillBlocks {
		from = to - s.maxBackfillBlocks + 1
		truncated = true
	}
	return from, to, missed, truncated
}

func (s *WebSocketSubscriber) sendBlock(ctx context.Context, out chan<- Block, block Block) error {
	if !s.shouldEmit(block) {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- block:
		if s.logger != nil {
			s.logger.Info("new block",
				"number", block.Number,
				"hash", block.Hash,
				"timestamp", block.Timestamp.Format(time.RFC3339),
			)
		}
	}
	return nil
}

func (s *WebSocketSubscriber) shouldEmit(block Block) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastBlock > 0 && block.Number <= s.lastBlock {
		return false
	}

	s.lastBlock = block.Number
	return true
}

// LastBlock returns the most recently emitted block number.
func (s *WebSocketSubscriber) LastBlock() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastBlock
}
