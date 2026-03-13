package security

import (
	"fmt"
	"sync"
	"testing"
)

func TestWatchdog_NoAnomalies(t *testing.T) {
	w := NewWatchdog(DefaultWatchdogThresholds())
	w.RecordAction("order", map[string]string{"symbol": "BTC/USD", "size": "100"})

	alerts := w.CheckAnomaly()
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts, got %d: %+v", len(alerts), alerts)
	}
}

func TestWatchdog_HighFrequency(t *testing.T) {
	thresholds := WatchdogThresholds{
		MaxOrdersPerMinute: 3,
		MaxOrderSize:       50000,
	}
	w := NewWatchdog(thresholds)

	for i := 0; i < 5; i++ {
		w.RecordAction("order", map[string]string{"symbol": "BTC/USD", "size": "100"})
	}

	alerts := w.CheckAnomaly()
	found := false
	for _, a := range alerts {
		if a.Type == "high_frequency" {
			found = true
			if a.Severity != SeverityWarning {
				t.Errorf("expected warning severity, got %s", a.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected high_frequency alert")
	}
}

func TestWatchdog_HighFrequencyCritical(t *testing.T) {
	thresholds := WatchdogThresholds{
		MaxOrdersPerMinute: 2,
	}
	w := NewWatchdog(thresholds)

	// More than 2x the limit -> critical
	for i := 0; i < 6; i++ {
		w.RecordAction("order", map[string]string{"symbol": "BTC/USD", "size": "10"})
	}

	alerts := w.CheckAnomaly()
	for _, a := range alerts {
		if a.Type == "high_frequency" && a.Severity != SeverityCritical {
			t.Errorf("expected critical severity for 6 orders with limit 2, got %s", a.Severity)
		}
	}
}

func TestWatchdog_LargeOrder(t *testing.T) {
	thresholds := WatchdogThresholds{
		MaxOrderSize: 1000,
	}
	w := NewWatchdog(thresholds)
	w.RecordAction("order", map[string]string{"symbol": "ETH/USD", "size": "1500"})

	alerts := w.CheckAnomaly()
	found := false
	for _, a := range alerts {
		if a.Type == "large_order" {
			found = true
			if a.Severity != SeverityWarning {
				t.Errorf("expected warning, got %s", a.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected large_order alert")
	}
}

func TestWatchdog_UnknownSymbol(t *testing.T) {
	thresholds := WatchdogThresholds{
		AllowedSymbols: map[string]bool{"BTC/USD": true, "ETH/USD": true},
	}
	w := NewWatchdog(thresholds)
	w.RecordAction("order", map[string]string{"symbol": "DOGE/USD", "size": "10"})

	alerts := w.CheckAnomaly()
	found := false
	for _, a := range alerts {
		if a.Type == "unknown_symbol" {
			found = true
			if a.Severity != SeverityCritical {
				t.Errorf("expected critical, got %s", a.Severity)
			}
		}
	}
	if !found {
		t.Fatal("expected unknown_symbol alert")
	}
}

func TestWatchdog_AllowedSymbolNoAlert(t *testing.T) {
	thresholds := WatchdogThresholds{
		AllowedSymbols: map[string]bool{"BTC/USD": true},
	}
	w := NewWatchdog(thresholds)
	w.RecordAction("order", map[string]string{"symbol": "BTC/USD", "size": "10"})

	alerts := w.CheckAnomaly()
	for _, a := range alerts {
		if a.Type == "unknown_symbol" {
			t.Fatal("should not flag allowed symbol")
		}
	}
}

func TestWatchdog_NonOrderActionsIgnored(t *testing.T) {
	thresholds := WatchdogThresholds{
		MaxOrdersPerMinute: 1,
		MaxOrderSize:       100,
		AllowedSymbols:     map[string]bool{"BTC/USD": true},
	}
	w := NewWatchdog(thresholds)

	// "view" actions should not count as orders
	for i := 0; i < 5; i++ {
		w.RecordAction("view", map[string]string{"symbol": "DOGE/USD", "size": "99999"})
	}

	alerts := w.CheckAnomaly()
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts for non-order actions, got %d", len(alerts))
	}
}

func TestWatchdog_ClearActions(t *testing.T) {
	w := NewWatchdog(DefaultWatchdogThresholds())
	w.RecordAction("order", map[string]string{})
	if w.ActionCount() != 1 {
		t.Fatalf("expected 1 action, got %d", w.ActionCount())
	}
	w.ClearActions()
	if w.ActionCount() != 0 {
		t.Fatalf("expected 0 actions after clear, got %d", w.ActionCount())
	}
}

func TestWatchdog_UpdateThresholds(t *testing.T) {
	w := NewWatchdog(DefaultWatchdogThresholds())
	w.RecordAction("order", map[string]string{"size": "100"})
	alerts := w.CheckAnomaly()
	if len(alerts) != 0 {
		t.Fatal("should have no alerts with default thresholds")
	}

	// Lower the max order size below 100
	w.UpdateThresholds(WatchdogThresholds{MaxOrderSize: 50})
	alerts = w.CheckAnomaly()
	found := false
	for _, a := range alerts {
		if a.Type == "large_order" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected large_order alert after threshold update")
	}
}

func TestWatchdog_ConcurrentAccess(t *testing.T) {
	w := NewWatchdog(DefaultWatchdogThresholds())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			w.RecordAction("order", map[string]string{
				"symbol": fmt.Sprintf("SYM%d", n),
				"size":   "10",
			})
			w.CheckAnomaly()
		}(i)
	}
	wg.Wait()

	if w.ActionCount() != 50 {
		t.Fatalf("expected 50 actions, got %d", w.ActionCount())
	}
}
