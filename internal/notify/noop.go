package notify

import "context"

// NoOpNotifier discards notifications.
type NoOpNotifier struct{}

func (NoOpNotifier) Notify(_ context.Context, _ string, _ bool) error {
	return nil
}
