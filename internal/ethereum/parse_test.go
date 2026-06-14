package ethereum

import (
	"testing"
	"time"
)

const sampleBlockNotification = `{
  "jsonrpc": "2.0",
  "method": "eth_subscription",
  "params": {
    "subscription": "0xabc",
    "result": {
      "number": "0x1163cc7",
      "hash": "0xdeadbeef",
      "timestamp": "0x65a4b2c0"
    }
  }
}`

func TestParseBlockFromSubscription(t *testing.T) {
	block, ok, err := parseBlockFromSubscription([]byte(sampleBlockNotification))
	if err != nil || !ok {
		t.Fatalf("parse failed: ok=%v err=%v", ok, err)
	}

	if block.Number != 18234567 {
		t.Fatalf("number: got %d want 18234567", block.Number)
	}
	if block.Hash != "0xdeadbeef" {
		t.Fatalf("hash: %s", block.Hash)
	}
	if block.Timestamp.IsZero() {
		t.Fatal("expected timestamp")
	}
}

func TestParseBlockFromSubscriptionIgnoresAck(t *testing.T) {
	_, ok, err := parseBlockFromSubscription([]byte(`{"jsonrpc":"2.0","id":1,"result":"0xsub"}`))
	if err != nil || ok {
		t.Fatalf("expected ignore ack, ok=%v err=%v", ok, err)
	}
}

func TestParseHexUint64(t *testing.T) {
	n, err := parseHexUint64("0x10")
	if err != nil || n != 16 {
		t.Fatalf("got %d err=%v", n, err)
	}
}

func TestReconnectDelayIncreasesWithJitter(t *testing.T) {
	d0 := reconnectDelay(0)
	d3 := reconnectDelay(3)
	if d0 < initialBackoff {
		t.Fatalf("too small: %v", d0)
	}
	if d3 <= d0 {
		t.Fatalf("expected backoff growth: d0=%v d3=%v", d0, d3)
	}
	if d3 > maxBackoff+maxBackoff/2 {
		t.Fatalf("backoff exceeded cap with jitter: %v", d3)
	}
}

func TestShouldEmitDedupAndGap(t *testing.T) {
	sub := &WebSocketSubscriber{logger: nil}

	b1 := Block{Number: 100, Timestamp: time.Now()}
	if !sub.shouldEmit(b1) {
		t.Fatal("first block should emit")
	}
	if sub.shouldEmit(b1) {
		t.Fatal("duplicate block should not emit")
	}

	b3 := Block{Number: 103, Timestamp: time.Now()}
	if !sub.shouldEmit(b3) {
		t.Fatal("new block should emit")
	}
	if sub.LastBlock() != 103 {
		t.Fatalf("last block: %d", sub.LastBlock())
	}
}
