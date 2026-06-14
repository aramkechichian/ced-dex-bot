package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestActionWebhookNotify(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewActionWebhook(srv.URL)
	if err := n.Notify(context.Background(), "hello arbitrage", true); err != nil {
		t.Fatal(err)
	}

	var payload ActionPayload
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Event != "arbitrage.opportunity" {
		t.Fatalf("event: %s", payload.Event)
	}
	if !payload.Simulated {
		t.Fatal("expected simulated")
	}
	if !strings.Contains(payload.Text, "hello arbitrage") {
		t.Fatalf("text: %s", payload.Text)
	}
}

func TestNoOpNotifier(t *testing.T) {
	if err := (NoOpNotifier{}).Notify(context.Background(), "x", false); err != nil {
		t.Fatal(err)
	}
}
