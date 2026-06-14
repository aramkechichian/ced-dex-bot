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

// WebSocketSubscriber streams newHeads from an Ethereum JSON-RPC WebSocket endpoint.
type WebSocketSubscriber struct {
	wsURL  string
	logger *slog.Logger
	mu     sync.Mutex
	lastBlock uint64
}

// NewWebSocketSubscriber creates a block streamer for eth_subscribe/newHeads.
func NewWebSocketSubscriber(wsURL string, logger *slog.Logger) *WebSocketSubscriber {
	return &WebSocketSubscriber{
		wsURL:  wsURL,
		logger: logger,
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
			// subscription confirmation or other rpc message
			var ack rpcMessage
			if err := json.Unmarshal(data, &ack); err == nil && ack.ID != nil {
				s.logger.Info("ethereum subscription confirmed")
			}
			continue
		}

		if !s.shouldEmit(block) {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- block:
			s.logger.Info("new block",
				"number", block.Number,
				"hash", block.Hash,
				"timestamp", block.Timestamp.Format(time.RFC3339),
			)
		}
	}
}

func (s *WebSocketSubscriber) shouldEmit(block Block) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastBlock == 0 {
		s.lastBlock = block.Number
		return true
	}

	if block.Number <= s.lastBlock {
		return false
	}

	if block.Number > s.lastBlock+1 {
		if s.logger != nil {
			s.logger.Warn("block gap detected",
				"last", s.lastBlock,
				"current", block.Number,
				"missed", block.Number-s.lastBlock-1,
				"error", common.ErrBlockGap,
			)
		}
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
