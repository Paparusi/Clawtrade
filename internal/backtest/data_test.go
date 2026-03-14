package backtest

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmp := t.TempDir()
	db, err := sql.Open("sqlite", tmp+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS candle_cache (
		symbol TEXT NOT NULL,
		timeframe TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		open REAL NOT NULL,
		high REAL NOT NULL,
		low REAL NOT NULL,
		close REAL NOT NULL,
		volume REAL NOT NULL,
		PRIMARY KEY (symbol, timeframe, timestamp)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// mockCandleAdapter implements adapter.TradingAdapter for testing.
type mockCandleAdapter struct {
	candles []adapter.Candle
}

func (m *mockCandleAdapter) GetCandles(_ context.Context, _, _ string, limit int) ([]adapter.Candle, error) {
	if limit > len(m.candles) {
		return m.candles, nil
	}
	return m.candles[len(m.candles)-limit:], nil
}

func (m *mockCandleAdapter) Name() string                                                  { return "mock" }
func (m *mockCandleAdapter) Capabilities() adapter.AdapterCaps                              { return adapter.AdapterCaps{} }
func (m *mockCandleAdapter) GetPrice(_ context.Context, _ string) (*adapter.Price, error)   { return nil, nil }
func (m *mockCandleAdapter) GetOrderBook(_ context.Context, _ string, _ int) (*adapter.OrderBook, error) { return nil, nil }
func (m *mockCandleAdapter) PlaceOrder(_ context.Context, _ adapter.Order) (*adapter.Order, error) { return nil, nil }
func (m *mockCandleAdapter) CancelOrder(_ context.Context, _ string) error                 { return nil }
func (m *mockCandleAdapter) GetOpenOrders(_ context.Context) ([]adapter.Order, error)      { return nil, nil }
func (m *mockCandleAdapter) GetBalances(_ context.Context) ([]adapter.Balance, error)      { return nil, nil }
func (m *mockCandleAdapter) GetPositions(_ context.Context) ([]adapter.Position, error)    { return nil, nil }
func (m *mockCandleAdapter) Connect(_ context.Context) error                               { return nil }
func (m *mockCandleAdapter) Disconnect() error                                             { return nil }
func (m *mockCandleAdapter) IsConnected() bool                                             { return false }

func TestDataLoader_FetchAndCache(t *testing.T) {
	db := setupTestDB(t)

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]adapter.Candle, 10)
	for i := range candles {
		candles[i] = adapter.Candle{
			Open:      100 + float64(i),
			High:      110 + float64(i),
			Low:       90 + float64(i),
			Close:     105 + float64(i),
			Volume:    1000 + float64(i),
			Timestamp: base.Add(time.Duration(i) * time.Minute),
		}
	}

	mock := &mockCandleAdapter{candles: candles}
	dl := NewDataLoader(db, mock)

	from := base
	to := base.Add(9 * time.Minute)

	ctx := context.Background()
	result, err := dl.LoadCandles(ctx, "BTC/USDT", "1m", from, to)
	if err != nil {
		t.Fatalf("LoadCandles failed: %v", err)
	}
	if len(result) != 10 {
		t.Fatalf("expected 10 candles, got %d", len(result))
	}

	// Verify cached: create new DataLoader with nil adapter
	dl2 := NewDataLoader(db, nil)
	result2, err := dl2.LoadCandles(ctx, "BTC/USDT", "1m", from, to)
	if err != nil {
		t.Fatalf("LoadCandles from cache failed: %v", err)
	}
	if len(result2) != 10 {
		t.Fatalf("expected 10 cached candles, got %d", len(result2))
	}

	// Verify candle data integrity
	for i, c := range result2 {
		if c.Open != candles[i].Open || c.Close != candles[i].Close {
			t.Errorf("candle %d mismatch: got open=%f close=%f, want open=%f close=%f",
				i, c.Open, c.Close, candles[i].Open, candles[i].Close)
		}
	}
}

func TestTimeframeToDuration(t *testing.T) {
	tests := []struct {
		tf   string
		want time.Duration
	}{
		{"1m", time.Minute},
		{"5m", 5 * time.Minute},
		{"15m", 15 * time.Minute},
		{"1h", time.Hour},
		{"4h", 4 * time.Hour},
		{"1d", 24 * time.Hour},
		{"1w", 168 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.tf, func(t *testing.T) {
			got := timeframeToDuration(tt.tf)
			if got != tt.want {
				t.Errorf("timeframeToDuration(%q) = %v, want %v", tt.tf, got, tt.want)
			}
		})
	}
}
