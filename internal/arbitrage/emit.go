package arbitrage

import (
	"context"
	"fmt"

	"github.com/aramik/ced-dex-bot/internal/notify"
)

// OpportunityEmitter prints and optionally notifies on detected opportunities.
type OpportunityEmitter struct {
	notifier notify.Notifier
	enabled  bool
}

// NewOpportunityEmitter creates an emitter. Notifications run when enabled and notifier is set.
func NewOpportunityEmitter(notifier notify.Notifier, notifyEnabled bool) OpportunityEmitter {
	return OpportunityEmitter{
		notifier: notifier,
		enabled:  notifyEnabled,
	}
}

// Emit prints the opportunity and sends notification if configured.
func (e OpportunityEmitter) Emit(ctx context.Context, opp *Opportunity, simulated bool) error {
	e.Print(opp, simulated)
	return e.Notify(ctx, opp, simulated)
}

// Print writes the opportunity to stdout.
func (e OpportunityEmitter) Print(opp *Opportunity, simulated bool) {
	if opp == nil {
		return
	}
	text := FormatOpportunity(opp)
	if simulated {
		text = "[SIMULATED]\n" + text
	}
	fmt.Println(text)
}

// Notify sends the opportunity alert to the configured notifier.
func (e OpportunityEmitter) Notify(ctx context.Context, opp *Opportunity, simulated bool) error {
	if opp == nil || !e.enabled || e.notifier == nil {
		return nil
	}

	var msg string
	if simulated {
		msg = "[SIMULATED] ARBITRAGE OPPORTUNITY\n" + FormatOpportunity(opp)
	} else {
		msg = "ARBITRAGE OPPORTUNITY DETECTED\n" + FormatOpportunity(opp)
	}

	if err := e.notifier.Notify(ctx, msg, simulated); err != nil {
		return fmt.Errorf("notify opportunity: %w", err)
	}
	return nil
}
