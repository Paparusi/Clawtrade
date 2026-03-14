package subagent

import (
	"sync"
	"time"
)

type EventBus struct {
	subscribers map[string][]chan Event
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan Event),
	}
}

func (b *EventBus) Subscribe(eventType string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 64)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

func (b *EventBus) Unsubscribe(eventType string, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[eventType]
	for i, s := range subs {
		if s == ch {
			b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

func (b *EventBus) Publish(ev Event) {
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers[ev.Type] {
		select {
		case ch <- ev:
		default:
			// drop if subscriber is full
		}
	}
}
