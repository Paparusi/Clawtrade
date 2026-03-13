package engine

// Built-in event type constants.
const (
	// Market events
	EventPriceUpdate     = "market.price.update"
	EventOrderBookUpdate = "market.orderbook.update"
	EventTradeExecution  = "market.trade.execution"

	// Trading events
	EventOrderPlaced    = "trade.order.placed"
	EventOrderFilled    = "trade.order.filled"
	EventOrderCancelled = "trade.order.cancelled"
	EventPositionOpened = "trade.position.opened"
	EventPositionClosed = "trade.position.closed"

	// Risk events
	EventRiskAlert      = "risk.alert"
	EventCircuitBreaker = "risk.circuit_breaker"
	EventDrawdownAlert  = "risk.drawdown"

	// System events
	EventAdapterConnected    = "system.adapter.connected"
	EventAdapterDisconnected = "system.adapter.disconnected"
	EventHealthCheck         = "system.health"

	// AI events
	EventAIResponse    = "ai.response"
	EventMemoryUpdated = "ai.memory.updated"
	EventSkillExecuted = "ai.skill.executed"
)

// EventFilter defines criteria for filtering events.
type EventFilter struct {
	Types    []string          `json:"types,omitempty"`    // specific event types
	Patterns []string          `json:"patterns,omitempty"` // wildcard patterns
	Fields   map[string]string `json:"fields,omitempty"`   // match specific data fields
}

// FilteredSubscriber wraps EventBus with advanced filtering.
type FilteredSubscriber struct {
	bus    *EventBus
	filter EventFilter
	subIDs []uint64
}

// NewFilteredSubscriber creates a subscriber with advanced filtering.
func NewFilteredSubscriber(bus *EventBus, filter EventFilter) *FilteredSubscriber {
	return &FilteredSubscriber{
		bus:    bus,
		filter: filter,
	}
}

// Subscribe registers handler with the filter.
// It subscribes to each explicit type and each wildcard pattern in the filter.
// If neither types nor patterns are specified, it subscribes to all events ("*").
// Field filtering is applied on top of pattern/type matching.
func (fs *FilteredSubscriber) Subscribe(handler EventHandler) {
	wrap := func(e Event) {
		if len(fs.filter.Fields) > 0 && !matchFields(e.Data, fs.filter.Fields) {
			return
		}
		handler(e)
	}

	if len(fs.filter.Types) == 0 && len(fs.filter.Patterns) == 0 {
		id := fs.bus.Subscribe("*", wrap)
		fs.subIDs = append(fs.subIDs, id)
		return
	}

	for _, t := range fs.filter.Types {
		id := fs.bus.Subscribe(t, wrap)
		fs.subIDs = append(fs.subIDs, id)
	}
	for _, p := range fs.filter.Patterns {
		id := fs.bus.Subscribe(p, wrap)
		fs.subIDs = append(fs.subIDs, id)
	}
}

// Unsubscribe removes all subscriptions.
func (fs *FilteredSubscriber) Unsubscribe() {
	for _, id := range fs.subIDs {
		fs.bus.Unsubscribe(id)
	}
	fs.subIDs = nil
}

// matchFields checks if event data matches all field filters.
func matchFields(data map[string]any, fields map[string]string) bool {
	for k, v := range fields {
		val, ok := data[k]
		if !ok {
			return false
		}
		// Convert the data value to string for comparison.
		str, ok := val.(string)
		if !ok {
			return false
		}
		if str != v {
			return false
		}
	}
	return true
}
