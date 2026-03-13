package security

import (
	"fmt"
	"sync"
	"time"
)

// AlertSeverity indicates how serious a watchdog alert is.
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// WatchdogAlert represents a single anomaly detected by the watchdog.
type WatchdogAlert struct {
	Type      string        `json:"type"`
	Severity  AlertSeverity `json:"severity"`
	Message   string        `json:"message"`
	Timestamp time.Time     `json:"timestamp"`
}

// WatchdogThresholds defines configurable limits for anomaly detection.
type WatchdogThresholds struct {
	MaxOrdersPerMinute int            // max orders allowed in a rolling 1-minute window
	MaxOrderSize       float64        // max order size (quote currency)
	AllowedSymbols     map[string]bool // if non-nil, only these symbols are allowed
}

// DefaultWatchdogThresholds returns sensible defaults.
func DefaultWatchdogThresholds() WatchdogThresholds {
	return WatchdogThresholds{
		MaxOrdersPerMinute: 10,
		MaxOrderSize:       50000,
		AllowedSymbols:     nil, // nil means all symbols allowed
	}
}

// actionRecord stores one recorded AI action.
type actionRecord struct {
	Action    string
	Details   map[string]string
	Timestamp time.Time
}

// Watchdog monitors AI behaviour and flags suspicious patterns.
type Watchdog struct {
	mu         sync.Mutex
	thresholds WatchdogThresholds
	actions    []actionRecord
}

// NewWatchdog creates a Watchdog with the given thresholds.
func NewWatchdog(thresholds WatchdogThresholds) *Watchdog {
	return &Watchdog{
		thresholds: thresholds,
		actions:    make([]actionRecord, 0),
	}
}

// RecordAction stores an AI action for later anomaly analysis.
func (w *Watchdog) RecordAction(action string, details map[string]string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.actions = append(w.actions, actionRecord{
		Action:    action,
		Details:   details,
		Timestamp: time.Now(),
	})
}

// UpdateThresholds replaces the current thresholds.
func (w *Watchdog) UpdateThresholds(t WatchdogThresholds) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.thresholds = t
}

// CheckAnomaly inspects all recorded actions and returns any alerts.
func (w *Watchdog) CheckAnomaly() []WatchdogAlert {
	w.mu.Lock()
	defer w.mu.Unlock()

	var alerts []WatchdogAlert
	now := time.Now()

	// --- Check 1: order frequency (rolling 1-minute window) ---
	oneMinuteAgo := now.Add(-1 * time.Minute)
	orderCount := 0
	for _, rec := range w.actions {
		if rec.Action == "order" && rec.Timestamp.After(oneMinuteAgo) {
			orderCount++
		}
	}
	if w.thresholds.MaxOrdersPerMinute > 0 && orderCount > w.thresholds.MaxOrdersPerMinute {
		severity := SeverityWarning
		if orderCount > w.thresholds.MaxOrdersPerMinute*2 {
			severity = SeverityCritical
		}
		alerts = append(alerts, WatchdogAlert{
			Type:     "high_frequency",
			Severity: severity,
			Message: fmt.Sprintf("%d orders in last minute exceeds limit of %d",
				orderCount, w.thresholds.MaxOrdersPerMinute),
			Timestamp: now,
		})
	}

	// --- Check 2: unusual order size ---
	for _, rec := range w.actions {
		if rec.Action != "order" {
			continue
		}
		sizeStr, ok := rec.Details["size"]
		if !ok {
			continue
		}
		var size float64
		fmt.Sscanf(sizeStr, "%f", &size)

		if w.thresholds.MaxOrderSize > 0 && size > w.thresholds.MaxOrderSize {
			severity := SeverityWarning
			if size > w.thresholds.MaxOrderSize*2 {
				severity = SeverityCritical
			}
			alerts = append(alerts, WatchdogAlert{
				Type:     "large_order",
				Severity: severity,
				Message: fmt.Sprintf("order size %.2f exceeds max %.2f",
					size, w.thresholds.MaxOrderSize),
				Timestamp: rec.Timestamp,
			})
		}
	}

	// --- Check 3: unusual symbols ---
	if w.thresholds.AllowedSymbols != nil {
		for _, rec := range w.actions {
			if rec.Action != "order" {
				continue
			}
			sym, ok := rec.Details["symbol"]
			if !ok {
				continue
			}
			if !w.thresholds.AllowedSymbols[sym] {
				alerts = append(alerts, WatchdogAlert{
					Type:      "unknown_symbol",
					Severity:  SeverityCritical,
					Message:   fmt.Sprintf("order on disallowed symbol %q", sym),
					Timestamp: rec.Timestamp,
				})
			}
		}
	}

	return alerts
}

// ClearActions removes all recorded actions (useful after processing).
func (w *Watchdog) ClearActions() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.actions = w.actions[:0]
}

// ActionCount returns the number of recorded actions.
func (w *Watchdog) ActionCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.actions)
}
