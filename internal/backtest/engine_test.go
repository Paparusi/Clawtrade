package backtest

import (
	"context"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	enginepkg "github.com/clawtrade/clawtrade/internal/engine"
)

// simpleTestStrategy buys after 3 consecutive up-candles, sells after 3 down.
type simpleTestStrategy struct{}

func (s *simpleTestStrategy) Evaluate(symbol string, candles []adapter.Candle) *BacktestSignal {
	n := len(candles)
	if n < 4 {
		return &BacktestSignal{Action: ActionHold}
	}
	ups, downs := 0, 0
	for i := n - 3; i < n; i++ {
		if candles[i].Close > candles[i-1].Close {
			ups++
		} else {
			downs++
		}
	}
	if ups == 3 {
		return &BacktestSignal{Action: ActionBuy, Strength: 0.8}
	}
	if downs == 3 {
		return &BacktestSignal{Action: ActionSell, Strength: 0.8}
	}
	return &BacktestSignal{Action: ActionHold}
}

func makeEngineCandles(n int, startPrice float64, priceFunc func(i int, prev float64) float64) []adapter.Candle {
	candles := make([]adapter.Candle, n)
	price := startPrice
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		if i > 0 {
			price = priceFunc(i, price)
		}
		candles[i] = adapter.Candle{
			Open:      price - 0.5,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000,
			Timestamp: base.Add(time.Duration(i) * time.Hour),
		}
	}
	return candles
}

func TestEngine_RunBasicBacktest(t *testing.T) {
	// Create 50 candles: uptrend (0-29: 100→158) then downtrend (30-49: 158→98)
	candles := makeEngineCandles(50, 100, func(i int, prev float64) float64 {
		if i < 30 {
			return prev + 2.0 // uptrend
		}
		return prev - 3.0 // downtrend
	})

	// Verify candle prices are as expected
	if candles[0].Close != 100 {
		t.Fatalf("expected first candle close=100, got %f", candles[0].Close)
	}
	if candles[29].Close != 158 {
		t.Fatalf("expected candle[29] close=158, got %f", candles[29].Close)
	}

	bus := enginepkg.NewEventBus()
	var progressCount atomic.Int64
	bus.Subscribe("backtest.progress", func(e enginepkg.Event) {
		progressCount.Add(1)
	})

	eng := &Engine{Bus: bus}
	cfg := BacktestConfig{
		Symbol:    "BTC/USDT",
		Timeframe: "1h",
		From:      candles[0].Timestamp,
		To:        candles[len(candles)-1].Timestamp,
		Capital:   10000,
		MakerFee:  0.001,
		TakerFee:  0.001,
		Slippage:  0.0005,
		TradeSize: 1.0,
	}

	result, err := eng.Run(context.Background(), cfg, candles, &simpleTestStrategy{})
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	// Some trades should have occurred
	if result.TotalTrades == 0 {
		t.Error("expected at least one trade")
	}

	// Equity curve should have 50 points
	if len(result.EquityCurve) != 50 {
		t.Errorf("expected 50 equity points, got %d", len(result.EquityCurve))
	}

	// Progress events should have been emitted
	// Give goroutines a moment to fire
	time.Sleep(50 * time.Millisecond)
	if progressCount.Load() == 0 {
		t.Error("expected progress events to be emitted")
	}

	// Basic sanity on result fields
	if result.Symbol != "BTC/USDT" {
		t.Errorf("expected symbol BTC/USDT, got %s", result.Symbol)
	}
	if result.Capital != 10000 {
		t.Errorf("expected capital 10000, got %f", result.Capital)
	}
	if result.FinalEquity <= 0 {
		t.Error("expected positive final equity")
	}

	t.Logf("Trades: %d, PnL: %.2f, Return: %.2f%%, MaxDD: %.4f, Sharpe: %.4f",
		result.TotalTrades, result.TotalPnL, result.TotalReturn, result.MaxDrawdown, result.SharpeRatio)
}

func TestEngine_Metrics(t *testing.T) {
	t.Run("SharpeRatio", func(t *testing.T) {
		// Known returns: mean=0.02, stddev should be calculable
		returns := []float64{0.01, 0.03, 0.02, 0.04, 0.00}
		sharpe := computeSharpeRatio(returns)
		if math.IsNaN(sharpe) || math.IsInf(sharpe, 0) {
			t.Errorf("expected finite Sharpe ratio, got %f", sharpe)
		}
		// Mean = 0.02, stddev ~ 0.01414
		// Sharpe = 0.02 / 0.01414 ~ 1.414
		if sharpe < 1.0 || sharpe > 2.0 {
			t.Errorf("expected Sharpe ratio around 1.4, got %f", sharpe)
		}
	})

	t.Run("SharpeRatio_ZeroStddev", func(t *testing.T) {
		returns := []float64{0.01, 0.01, 0.01}
		sharpe := computeSharpeRatio(returns)
		// With zero stddev, should return 0
		if sharpe != 0 {
			t.Errorf("expected Sharpe=0 for constant returns, got %f", sharpe)
		}
	})

	t.Run("SharpeRatio_Empty", func(t *testing.T) {
		sharpe := computeSharpeRatio(nil)
		if sharpe != 0 {
			t.Errorf("expected Sharpe=0 for empty returns, got %f", sharpe)
		}
	})

	t.Run("MaxDrawdown", func(t *testing.T) {
		equity := []float64{100, 110, 105, 120, 90, 115}
		dd := computeMaxDrawdown(equity)
		// Peak=120, trough=90, maxDD = 30/120 = 0.25
		expected := 30.0 / 120.0
		if math.Abs(dd-expected) > 1e-9 {
			t.Errorf("expected maxDD=%f, got %f", expected, dd)
		}
	})

	t.Run("MaxDrawdown_NoDrawdown", func(t *testing.T) {
		equity := []float64{100, 110, 120, 130}
		dd := computeMaxDrawdown(equity)
		if dd != 0 {
			t.Errorf("expected maxDD=0 for monotonically increasing, got %f", dd)
		}
	})

	t.Run("MaxDrawdown_Empty", func(t *testing.T) {
		dd := computeMaxDrawdown(nil)
		if dd != 0 {
			t.Errorf("expected maxDD=0 for empty, got %f", dd)
		}
	})
}
