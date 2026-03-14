package streaming

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/subagent"
)

// Bridge connects subagent.EventBus to engine.EventBus, forwarding
// sub-agent events with "agent.*" prefix for WebSocket broadcast.
type Bridge struct {
	saBus  *subagent.EventBus
	engBus *engine.EventBus
	cancel context.CancelFunc
}

// NewBridge creates a Bridge that forwards events from saBus to engBus.
func NewBridge(saBus *subagent.EventBus, engBus *engine.EventBus) *Bridge {
	return &Bridge{saBus: saBus, engBus: engBus}
}

var bridgedTypes = []string{"analysis", "counter_analysis", "narrative", "reflection", "correlation"}

// Start begins forwarding events from the subagent bus to the engine bus.
func (b *Bridge) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	for _, et := range bridgedTypes {
		ch := b.saBus.Subscribe(et)
		go b.forwardLoop(ctx, ch)
	}
}

// Stop cancels all forwarding goroutines.
func (b *Bridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
}

func (b *Bridge) forwardLoop(ctx context.Context, ch <-chan subagent.Event) {
	for {
		select {
		case ev := <-ch:
			engEvent := engine.Event{
				Type: mapEventType(ev.Type),
				Data: map[string]any{
					"source":  ev.Source,
					"symbol":  ev.Symbol,
					"summary": generateSummary(ev),
					"data":    ev.Data,
				},
			}
			b.engBus.Publish(engEvent)
		case <-ctx.Done():
			return
		}
	}
}

func mapEventType(saType string) string {
	switch saType {
	case "analysis":
		return "agent.analysis"
	case "counter_analysis":
		return "agent.counter"
	case "narrative":
		return "agent.narrative"
	case "reflection":
		return "agent.reflection"
	case "correlation":
		return "agent.correlation"
	default:
		return "agent." + saType
	}
}

func generateSummary(ev subagent.Event) string {
	source := ev.Source
	symbol := ev.Symbol

	// Try to extract key insight from data
	if ev.Data != nil {
		if synthesis, ok := ev.Data["synthesis"].(string); ok {
			if symbol != "" {
				return fmt.Sprintf("%s: %s — %s", source, symbol, truncate(synthesis, 80))
			}
			return fmt.Sprintf("%s: %s", source, truncate(synthesis, 80))
		}
		if counter, ok := ev.Data["counter"].(string); ok {
			return fmt.Sprintf("%s: %s — %s", source, symbol, truncate(counter, 80))
		}
	}

	if symbol != "" {
		return fmt.Sprintf("%s: %s update", source, symbol)
	}
	return fmt.Sprintf("%s: update", source)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
