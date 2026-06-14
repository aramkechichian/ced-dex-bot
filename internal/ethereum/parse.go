package ethereum

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

const (
	readTimeout       = 30 * time.Second
	initialBackoff    = time.Second
	maxBackoff        = 60 * time.Second
	maxBackoffAttempt = 6
)

type subscribeRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Params  *subscriptionParams `json:"params,omitempty"`
}

type subscriptionParams struct {
	Subscription string          `json:"subscription"`
	Result       blockHeaderJSON `json:"result"`
}

type blockHeaderJSON struct {
	Number    string `json:"number"`
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
}

func newSubscribeMessage() subscribeRequest {
	return subscribeRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_subscribe",
		Params:  []interface{}{"newHeads"},
	}
}

func parseBlockHeader(raw blockHeaderJSON) (Block, error) {
	number, err := parseHexUint64(raw.Number)
	if err != nil {
		return Block{}, fmt.Errorf("parse block number: %w", err)
	}

	ts, err := parseHexUint64(raw.Timestamp)
	if err != nil {
		return Block{}, fmt.Errorf("parse block timestamp: %w", err)
	}

	return Block{
		Number:    number,
		Hash:      raw.Hash,
		Timestamp: time.Unix(int64(ts), 0).UTC(),
	}, nil
}

func parseBlockFromSubscription(data []byte) (Block, bool, error) {
	var msg rpcMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return Block{}, false, fmt.Errorf("unmarshal rpc message: %w", err)
	}

	if msg.Method != "eth_subscription" || msg.Params == nil {
		return Block{}, false, nil
	}

	block, err := parseBlockHeader(msg.Params.Result)
	if err != nil {
		return Block{}, false, err
	}

	return block, true, nil
}

func parseHexUint64(hex string) (uint64, error) {
	hex = strings.TrimSpace(hex)
	if hex == "" {
		return 0, fmt.Errorf("empty hex string")
	}
	return strconv.ParseUint(strings.TrimPrefix(hex, "0x"), 16, 64)
}

func reconnectDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > maxBackoffAttempt {
		attempt = maxBackoffAttempt
	}

	delay := initialBackoff * time.Duration(1<<attempt)
	if delay > maxBackoff {
		delay = maxBackoff
	}

	jitter := time.Duration(rand.Int64N(int64(delay / 2)))
	return delay + jitter
}
