package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const actionWebhookTimeout = 10 * time.Second

// ActionWebhook posts opportunity alerts to a generic HTTP webhook.
// Compatible with Slack incoming webhooks (uses "text" field) and custom action APIs.
type ActionWebhook struct {
	webhookURL string
	client     *http.Client
}

// NewActionWebhook creates a generic action webhook notifier.
func NewActionWebhook(webhookURL string) *ActionWebhook {
	return &ActionWebhook{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: actionWebhookTimeout},
	}
}

// ActionPayload is the JSON body sent to ACTION_WEBHOOK_URL.
// External services (Slack, n8n, custom executors) can consume event + text + data.
type ActionPayload struct {
	Event     string `json:"event"`
	Simulated bool   `json:"simulated"`
	Text      string `json:"text"`
}

// Notify POSTs the payload to the configured webhook URL.
func (a *ActionWebhook) Notify(ctx context.Context, message string, simulated bool) error {
	if a == nil || a.webhookURL == "" {
		return fmt.Errorf("action webhook url is required")
	}

	body, err := json.Marshal(ActionPayload{
		Event:     "arbitrage.opportunity",
		Simulated: simulated,
		Text:      message,
	})
	if err != nil {
		return fmt.Errorf("marshal action payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create action webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("action webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("action webhook status %d", resp.StatusCode)
	}

	return nil
}
