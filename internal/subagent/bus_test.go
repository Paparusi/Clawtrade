package subagent

import (
	"testing"
	"time"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe("analysis")

	go func() {
		bus.Publish(Event{
			Type:   "analysis",
			Source: "test",
			Symbol: "BTC/USDT",
			Data:   map[string]any{"bias": "bullish"},
		})
	}()

	select {
	case ev := <-ch:
		if ev.Type != "analysis" {
			t.Errorf("expected type 'analysis', got %q", ev.Type)
		}
		if ev.Symbol != "BTC/USDT" {
			t.Errorf("expected symbol 'BTC/USDT', got %q", ev.Symbol)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe("alert")
	ch2 := bus.Subscribe("alert")

	bus.Publish(Event{Type: "alert", Source: "test"})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case ev := <-ch:
			if ev.Type != "alert" {
				t.Error("wrong event type")
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe("analysis")
	bus.Unsubscribe("analysis", ch)

	bus.Publish(Event{Type: "analysis", Source: "test"})

	select {
	case <-ch:
		t.Fatal("should not receive after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}
