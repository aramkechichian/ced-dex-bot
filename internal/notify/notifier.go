package notify

import "context"

// Notifier sends alerts when an arbitrage opportunity is detected.
type Notifier interface {
	Notify(ctx context.Context, message string, simulated bool) error
}
