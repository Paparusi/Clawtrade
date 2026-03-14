package alert

import (
	"strings"
	"testing"
)

func TestFormatBriefing(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	store.Create(&Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true})
	store.LogTrigger(1, "price.update", 71000, "BTC above 70k")

	history, _ := store.TodayHistory()

	msg := FormatBriefing(10500.0, 3, len(history))
	if msg == "" {
		t.Fatal("expected non-empty briefing")
	}
	if !strings.Contains(msg, "10,500") {
		t.Errorf("expected portfolio value in briefing, got: %s", msg)
	}
	if !strings.Contains(msg, "Active Alerts: 3") {
		t.Errorf("expected active alerts count, got: %s", msg)
	}
	if !strings.Contains(msg, "Triggered Today: 1") {
		t.Errorf("expected triggered today count, got: %s", msg)
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{500.0, "500.00"},
		{10500.0, "10,500.00"},
		{1500000.0, "1.50M"},
	}
	for _, tt := range tests {
		got := formatMoney(tt.value)
		if got != tt.expected {
			t.Errorf("formatMoney(%.2f) = %s, want %s", tt.value, got, tt.expected)
		}
	}
}
