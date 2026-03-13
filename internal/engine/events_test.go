package engine

import (
	"sync"
	"testing"
	"time"
)

func TestFilteredSubscriber_TypeFilter(t *testing.T) {
	bus := NewEventBus()

	var received []Event
	var mu sync.Mutex

	fs := NewFilteredSubscriber(bus, EventFilter{
		Types: []string{EventOrderFilled, EventOrderPlaced},
	})

	fs.Subscribe(func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	bus.Publish(Event{Type: EventOrderFilled, Data: map[string]any{"id": "1"}})
	bus.Publish(Event{Type: EventOrderPlaced, Data: map[string]any{"id": "2"}})
	bus.Publish(Event{Type: EventOrderCancelled, Data: map[string]any{"id": "3"}})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
}

func TestFilteredSubscriber_PatternFilter(t *testing.T) {
	bus := NewEventBus()

	var received []Event
	var mu sync.Mutex

	fs := NewFilteredSubscriber(bus, EventFilter{
		Patterns: []string{"market.*"},
	})

	fs.Subscribe(func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	bus.Publish(Event{Type: EventPriceUpdate})
	bus.Publish(Event{Type: EventOrderBookUpdate})
	bus.Publish(Event{Type: EventTradeExecution})
	bus.Publish(Event{Type: EventOrderFilled}) // should NOT match

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 3 {
		t.Fatalf("expected 3 market events, got %d", len(received))
	}
}

func TestFilteredSubscriber_FieldFilter(t *testing.T) {
	bus := NewEventBus()

	var received []Event
	var mu sync.Mutex

	fs := NewFilteredSubscriber(bus, EventFilter{
		Patterns: []string{"trade.*"},
		Fields:   map[string]string{"symbol": "BTC/USDT"},
	})

	fs.Subscribe(func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	bus.Publish(Event{Type: EventOrderFilled, Data: map[string]any{"symbol": "BTC/USDT"}})
	bus.Publish(Event{Type: EventOrderFilled, Data: map[string]any{"symbol": "ETH/USDT"}})
	bus.Publish(Event{Type: EventOrderPlaced, Data: map[string]any{"symbol": "BTC/USDT"}})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 events matching field filter, got %d", len(received))
	}
}

func TestMatchFields(t *testing.T) {
	tests := []struct {
		name   string
		data   map[string]any
		fields map[string]string
		want   bool
	}{
		{
			name:   "exact match",
			data:   map[string]any{"symbol": "BTC/USDT", "side": "buy"},
			fields: map[string]string{"symbol": "BTC/USDT"},
			want:   true,
		},
		{
			name:   "multiple fields match",
			data:   map[string]any{"symbol": "BTC/USDT", "side": "buy"},
			fields: map[string]string{"symbol": "BTC/USDT", "side": "buy"},
			want:   true,
		},
		{
			name:   "field mismatch",
			data:   map[string]any{"symbol": "ETH/USDT"},
			fields: map[string]string{"symbol": "BTC/USDT"},
			want:   false,
		},
		{
			name:   "missing field",
			data:   map[string]any{"price": "100"},
			fields: map[string]string{"symbol": "BTC/USDT"},
			want:   false,
		},
		{
			name:   "non-string value",
			data:   map[string]any{"count": 42},
			fields: map[string]string{"count": "42"},
			want:   false,
		},
		{
			name:   "empty fields always matches",
			data:   map[string]any{"anything": "value"},
			fields: map[string]string{},
			want:   true,
		},
		{
			name:   "nil data fails",
			data:   nil,
			fields: map[string]string{"key": "val"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchFields(tt.data, tt.fields)
			if got != tt.want {
				t.Errorf("matchFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilteredSubscriber_Unsubscribe(t *testing.T) {
	bus := NewEventBus()

	var count int
	var mu sync.Mutex

	fs := NewFilteredSubscriber(bus, EventFilter{
		Types: []string{EventRiskAlert},
	})

	fs.Subscribe(func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	bus.Publish(Event{Type: EventRiskAlert})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Fatalf("expected 1 event before unsub, got %d", count)
	}
	mu.Unlock()

	fs.Unsubscribe()

	bus.Publish(Event{Type: EventRiskAlert})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 event after unsub, got %d", count)
	}
}
