package streaming

import (
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/subagent"
)

func TestBridge_ForwardsAnalysisEvent(t *testing.T) {
	saBus := subagent.NewEventBus()
	engBus := engine.NewEventBus()

	b := NewBridge(saBus, engBus)
	b.Start()
	defer b.Stop()

	received := make(chan engine.Event, 1)
	engBus.Subscribe("agent.analysis", func(e engine.Event) {
		received <- e
	})

	saBus.Publish(subagent.Event{
		Type:   "analysis",
		Source: "market-analyst",
		Symbol: "BTC/USDT",
		Data:   map[string]any{"synthesis": "bullish 78%"},
	})

	select {
	case ev := <-received:
		if ev.Type != "agent.analysis" {
			t.Errorf("expected type 'agent.analysis', got %q", ev.Type)
		}
		data := ev.Data
		if data["source"] != "market-analyst" {
			t.Errorf("expected source 'market-analyst', got %v", data["source"])
		}
		if data["symbol"] != "BTC/USDT" {
			t.Errorf("expected symbol 'BTC/USDT', got %v", data["symbol"])
		}
		if _, ok := data["summary"]; !ok {
			t.Error("expected summary field")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for bridged event")
	}
}

func TestBridge_MapsEventTypes(t *testing.T) {
	tests := []struct {
		saType  string
		engType string
	}{
		{"analysis", "agent.analysis"},
		{"counter_analysis", "agent.counter"},
		{"narrative", "agent.narrative"},
		{"reflection", "agent.reflection"},
		{"correlation", "agent.correlation"},
	}
	for _, tt := range tests {
		got := mapEventType(tt.saType)
		if got != tt.engType {
			t.Errorf("mapEventType(%q) = %q, want %q", tt.saType, got, tt.engType)
		}
	}
}

func TestBridge_GeneratesSummary(t *testing.T) {
	ev := subagent.Event{
		Source: "market-analyst",
		Symbol: "BTC/USDT",
		Data:   map[string]any{"synthesis": "bullish with 78% confidence"},
	}
	summary := generateSummary(ev)
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}
