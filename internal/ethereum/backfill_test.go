package ethereum

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type stubBlockFetcher struct {
	blocks map[uint64]Block
	errAt  uint64
}

func (s *stubBlockFetcher) FetchBlock(_ context.Context, number uint64) (Block, error) {
	if number == s.errAt {
		return Block{}, fmt.Errorf("fetch failed")
	}
	b, ok := s.blocks[number]
	if !ok {
		return Block{}, fmt.Errorf("block %d not found", number)
	}
	return b, nil
}

func TestEmitBlockBackfillsGap(t *testing.T) {
	sub := NewWebSocketSubscriber("", nil)
	sub.fetcher = &stubBlockFetcher{
		blocks: map[uint64]Block{
			101: {Number: 101, Hash: "0x101", Timestamp: time.Unix(101, 0).UTC()},
			102: {Number: 102, Hash: "0x102", Timestamp: time.Unix(102, 0).UTC()},
		},
	}
	sub.maxBackfillBlocks = 10

	out := make(chan Block, 8)
	ctx := context.Background()

	sub.lastBlock = 100

	if err := sub.emitBlock(ctx, out, Block{Number: 103, Hash: "0x103", Timestamp: time.Unix(103, 0).UTC()}); err != nil {
		t.Fatal(err)
	}

	want := []uint64{101, 102, 103}
	for i, n := range want {
		select {
		case b := <-out:
			if b.Number != n {
				t.Fatalf("block %d: got %d", i, b.Number)
			}
		default:
			t.Fatalf("missing block %d", n)
		}
	}
}

func TestBackfillTruncatesLargeGap(t *testing.T) {
	sub := NewWebSocketSubscriber("", nil)
	sub.lastBlock = 100
	sub.maxBackfillBlocks = 2

	from, to, _, truncated := sub.backfillRange(100, 120)
	if from != 118 || to != 119 || !truncated {
		t.Fatalf("range: from=%d to=%d truncated=%v", from, to, truncated)
	}
}
