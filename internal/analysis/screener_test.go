package analysis

import (
	"math"
	"testing"
)

// makeCandles generates n candles with linearly increasing close prices
// starting at startPrice with the given step.
func makeCandles(n int, startPrice, step float64) []Candle {
	candles := make([]Candle, n)
	for i := 0; i < n; i++ {
		p := startPrice + float64(i)*step
		candles[i] = Candle{
			Open:      p - 0.5,
			High:      p + 1.0,
			Low:       p - 1.0,
			Close:     p,
			Volume:    1000 + float64(i)*100,
			Timestamp: int64(i),
		}
	}
	return candles
}

// makeCandlesFromCloses creates candles from a slice of close prices.
func makeCandlesFromCloses(closes []float64) []Candle {
	candles := make([]Candle, len(closes))
	for i, c := range closes {
		candles[i] = Candle{
			Open:      c,
			High:      c + 1,
			Low:       c - 1,
			Close:     c,
			Volume:    1000,
			Timestamp: int64(i),
		}
	}
	return candles
}

func TestScreenerRSIAbove(t *testing.T) {
	// Generate strongly rising candles so RSI is high.
	candles := makeCandles(50, 10, 2.0)
	s := NewScreener([]ScreenerCondition{
		{Indicator: "rsi", Operator: "above", Value: 50, Period: 14},
	})
	result := s.Screen("AAPL", candles)
	if !result.PassedAll {
		t.Errorf("expected RSI above 50 to pass for rising candles, got value=%f", result.Conditions[0].Value)
	}
}

func TestScreenerRSIBelow(t *testing.T) {
	// Generate strongly falling candles so RSI is low.
	candles := makeCandles(50, 200, -2.0)
	s := NewScreener([]ScreenerCondition{
		{Indicator: "rsi", Operator: "below", Value: 50, Period: 14},
	})
	result := s.Screen("AAPL", candles)
	if !result.PassedAll {
		t.Errorf("expected RSI below 50 to pass for falling candles, got value=%f", result.Conditions[0].Value)
	}
}

func TestScreenerSMACrossesAbove(t *testing.T) {
	// Build data where the SMA(5) is below a threshold then rises above it.
	// Start with low prices, then jump up.
	closes := make([]float64, 30)
	for i := 0; i < 20; i++ {
		closes[i] = 10.0
	}
	// Prices jump, causing SMA(5) to cross above 10.
	for i := 20; i < 30; i++ {
		closes[i] = 20.0
	}
	candles := makeCandlesFromCloses(closes)

	s := NewScreener([]ScreenerCondition{
		{Indicator: "sma", Operator: "crosses_above", Value: 12.0, Period: 5},
	})
	result := s.Screen("TSLA", candles)
	// The SMA(5) should have crossed above 12 at some transition point,
	// but we check the last two values. The last SMA(5) = 20, second-to-last = 20.
	// We need the crossing to happen at the last value specifically.
	// Let's check that the condition was evaluated without error.
	if len(result.Conditions) == 0 {
		t.Fatal("expected at least one condition result")
	}
	// Build a scenario where crossing happens at the last candle.
	closes2 := make([]float64, 10)
	for i := 0; i < 8; i++ {
		closes2[i] = 10.0
	}
	closes2[8] = 10.0
	closes2[9] = 22.0 // This pushes the SMA(5) from ~10 to 12.4
	candles2 := makeCandlesFromCloses(closes2)
	s2 := NewScreener([]ScreenerCondition{
		{Indicator: "sma", Operator: "crosses_above", Value: 12.0, Period: 5},
	})
	result2 := s2.Screen("TSLA", candles2)
	if !result2.PassedAll {
		t.Errorf("expected SMA crosses_above to pass, got value=%f passed=%v",
			result2.Conditions[0].Value, result2.Conditions[0].Passed)
	}
}

func TestScreenerPriceBetween(t *testing.T) {
	candles := makeCandles(20, 50, 0.5)
	lastClose := candles[len(candles)-1].Close // 50 + 19*0.5 = 59.5

	s := NewScreener([]ScreenerCondition{
		{Indicator: "price", Operator: "between", Value: 55, Value2: 65},
	})
	result := s.Screen("GOOG", candles)
	if !result.PassedAll {
		t.Errorf("expected price between 55-65 to pass, lastClose=%f, value=%f",
			lastClose, result.Conditions[0].Value)
	}

	// Test that price outside range fails.
	s2 := NewScreener([]ScreenerCondition{
		{Indicator: "price", Operator: "between", Value: 100, Value2: 200},
	})
	result2 := s2.Screen("GOOG", candles)
	if result2.PassedAll {
		t.Error("expected price between 100-200 to fail")
	}
}

func TestScreenerVolumeFilter(t *testing.T) {
	candles := makeCandles(20, 50, 1.0)
	// Last candle volume = 1000 + 19*100 = 2900
	s := NewScreener([]ScreenerCondition{
		{Indicator: "volume", Operator: "above", Value: 2000},
	})
	result := s.Screen("MSFT", candles)
	if !result.PassedAll {
		t.Errorf("expected volume above 2000 to pass, got value=%f", result.Conditions[0].Value)
	}
}

func TestScreenerMultipleConditionsAND(t *testing.T) {
	// Rising candles: RSI should be high, price should be high, volume should be set.
	candles := makeCandles(50, 10, 2.0)
	lastClose := candles[len(candles)-1].Close // 10 + 49*2 = 108

	s := NewScreener([]ScreenerCondition{
		{Indicator: "rsi", Operator: "above", Value: 50, Period: 14},
		{Indicator: "price", Operator: "above", Value: 100},
		{Indicator: "volume", Operator: "above", Value: 5000},
	})
	result := s.Screen("AAPL", candles)
	if !result.PassedAll {
		t.Errorf("expected all conditions to pass: lastClose=%f", lastClose)
		for i, cr := range result.Conditions {
			t.Logf("  condition %d (%s %s): passed=%v value=%f",
				i, cr.Condition.Indicator, cr.Condition.Operator, cr.Passed, cr.Value)
		}
	}

	// Now add a condition that should fail: price below 50.
	s2 := NewScreener([]ScreenerCondition{
		{Indicator: "rsi", Operator: "above", Value: 50, Period: 14},
		{Indicator: "price", Operator: "below", Value: 50},
	})
	result2 := s2.Screen("AAPL", candles)
	if result2.PassedAll {
		t.Error("expected AND logic to fail when one condition fails")
	}
}

func TestScreenMultipleSymbols(t *testing.T) {
	data := map[string][]Candle{
		"SYM_A": makeCandles(50, 10, 2.0),  // rising
		"SYM_B": makeCandles(50, 200, -2.0), // falling
		"SYM_C": makeCandles(50, 50, 0.5),  // gently rising
	}

	s := NewScreener([]ScreenerCondition{
		{Indicator: "rsi", Operator: "above", Value: 60, Period: 14},
	})

	results := s.ScreenMultiple(data)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check each symbol appears.
	found := make(map[string]ScreenerResult)
	for _, r := range results {
		found[r.Symbol] = r
	}
	for _, sym := range []string{"SYM_A", "SYM_B", "SYM_C"} {
		if _, ok := found[sym]; !ok {
			t.Errorf("missing result for %s", sym)
		}
	}

	// SYM_B (falling) should not have RSI above 60.
	if found["SYM_B"].PassedAll {
		t.Errorf("expected SYM_B (falling) to fail RSI above 60, got value=%f",
			found["SYM_B"].Conditions[0].Value)
	}
}

func TestScreenerScoreCalculation(t *testing.T) {
	candles := makeCandles(50, 10, 2.0)
	lastClose := candles[len(candles)-1].Close // 108

	s := NewScreener([]ScreenerCondition{
		{Indicator: "price", Operator: "above", Value: 100},  // pass (108 > 100)
		{Indicator: "price", Operator: "above", Value: 50},   // pass (108 > 50)
		{Indicator: "price", Operator: "below", Value: 50},   // fail (108 < 50 is false)
	})
	result := s.Screen("AAPL", candles)
	expectedScore := 2.0 / 3.0

	if math.Abs(result.Score-expectedScore) > 0.01 {
		t.Errorf("expected score ~%.3f, got %.3f", expectedScore, result.Score)
	}
	if result.PassedAll {
		t.Error("expected PassedAll to be false when one condition fails")
	}
	_ = lastClose
}

func TestScreenerInsufficientData(t *testing.T) {
	// Only 3 candles, not enough for RSI(14).
	candles := makeCandles(3, 10, 1.0)
	s := NewScreener([]ScreenerCondition{
		{Indicator: "rsi", Operator: "above", Value: 50, Period: 14},
	})
	result := s.Screen("TINY", candles)
	// Should not crash; condition should fail.
	if result.PassedAll {
		t.Error("expected insufficient data to cause condition to fail")
	}
	if len(result.Conditions) != 1 {
		t.Fatalf("expected 1 condition result, got %d", len(result.Conditions))
	}
	if result.Conditions[0].Passed {
		t.Error("expected condition to not pass with insufficient data")
	}
}

func TestScreenerEmptyCandles(t *testing.T) {
	s := NewScreener([]ScreenerCondition{
		{Indicator: "price", Operator: "above", Value: 50},
	})
	result := s.Screen("EMPTY", nil)
	if result.PassedAll {
		t.Error("expected empty candles to fail")
	}
}

func TestScreenerBollingerCondition(t *testing.T) {
	// Generate enough data for Bollinger Bands (period 20).
	candles := makeCandles(30, 50, 0.5)
	s := NewScreener([]ScreenerCondition{
		{Indicator: "bollinger", Operator: "between", Value: 0.0, Value2: 1.0, Period: 20},
	})
	result := s.Screen("BB", candles)
	if len(result.Conditions) == 0 {
		t.Fatal("expected condition result")
	}
	// With steadily rising prices, %B should be defined and within a reasonable range.
	val := result.Conditions[0].Value
	if math.IsNaN(val) {
		t.Error("expected a valid Bollinger %B value, got NaN")
	}
}

func TestScreenerMACDCondition(t *testing.T) {
	candles := makeCandles(50, 10, 1.0)
	s := NewScreener([]ScreenerCondition{
		{Indicator: "macd", Operator: "above", Value: 0},
	})
	result := s.Screen("MACD_TEST", candles)
	if len(result.Conditions) == 0 {
		t.Fatal("expected condition result")
	}
	// With steadily rising prices, MACD line should be positive.
	if !result.Conditions[0].Passed {
		t.Errorf("expected MACD above 0 for rising prices, got value=%f",
			result.Conditions[0].Value)
	}
}
