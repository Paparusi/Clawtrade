package analysis

import (
	"math"
	"testing"
)

// helper: create candles with given close prices (open/high/low = close).
func closesOnly(closes []float64) []Candle {
	candles := make([]Candle, len(closes))
	for i, c := range closes {
		candles[i] = Candle{Open: c, High: c, Low: c, Close: c, Timestamp: int64(i)}
	}
	return candles
}

// helper: create candles with explicit high/low/close.
func hlcCandles(data [][3]float64) []Candle {
	candles := make([]Candle, len(data))
	for i, d := range data {
		candles[i] = Candle{
			Open:      d[2], // use close as open
			High:      d[0],
			Low:       d[1],
			Close:     d[2],
			Timestamp: int64(i),
		}
	}
	return candles
}

func approxEqual(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}
	return math.Abs(a-b) < tol
}

// --- SMA Tests ---

func TestSMA_Basic(t *testing.T) {
	candles := closesOnly([]float64{1, 2, 3, 4, 5})
	result := SMA(candles, 3)

	if len(result) != 5 {
		t.Fatalf("expected length 5, got %d", len(result))
	}
	// First two should be NaN.
	if !math.IsNaN(result[0]) {
		t.Errorf("result[0] should be NaN, got %f", result[0])
	}
	if !math.IsNaN(result[1]) {
		t.Errorf("result[1] should be NaN, got %f", result[1])
	}
	// SMA values.
	expected := []float64{2, 3, 4}
	for i, exp := range expected {
		if !approxEqual(result[i+2], exp, 1e-9) {
			t.Errorf("result[%d] = %f, want %f", i+2, result[i+2], exp)
		}
	}
}

func TestSMA_Period1(t *testing.T) {
	candles := closesOnly([]float64{5, 10, 15})
	result := SMA(candles, 1)
	for i, c := range candles {
		if !approxEqual(result[i], c.Close, 1e-9) {
			t.Errorf("SMA period=1: result[%d] = %f, want %f", i, result[i], c.Close)
		}
	}
}

// --- EMA Tests ---

func TestEMA_FirstValueEqualsSMA(t *testing.T) {
	candles := closesOnly([]float64{22, 24, 23, 25, 26, 28, 27, 29, 30, 31})
	period := 5
	ema := EMA(candles, period)
	sma := SMA(candles, period)

	// First period-1 values should be NaN.
	for i := 0; i < period-1; i++ {
		if !math.IsNaN(ema[i]) {
			t.Errorf("EMA[%d] should be NaN, got %f", i, ema[i])
		}
	}
	// Value at period-1 should equal SMA.
	if !approxEqual(ema[period-1], sma[period-1], 1e-9) {
		t.Errorf("EMA[%d] = %f, want SMA = %f", period-1, ema[period-1], sma[period-1])
	}
}

func TestEMA_WeightsRecentMore(t *testing.T) {
	// After seed, a jump in price should pull EMA up faster than SMA.
	candles := closesOnly([]float64{10, 10, 10, 10, 10, 20, 20, 20})
	period := 5
	ema := EMA(candles, period)
	sma := SMA(candles, period)

	// At index 5 (first value after jump), EMA should be > SMA because EMA
	// weights the new 20 more heavily.
	if ema[5] <= sma[5] {
		t.Errorf("EMA[5]=%f should be > SMA[5]=%f after price jump", ema[5], sma[5])
	}
}

// --- RSI Tests ---

func TestRSI_Uptrend(t *testing.T) {
	// Monotonically increasing closes.
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	candles := closesOnly(closes)
	rsi := RSI(candles, 14)

	// First 14 values should be NaN.
	for i := 0; i < 14; i++ {
		if !math.IsNaN(rsi[i]) {
			t.Errorf("RSI[%d] should be NaN in uptrend, got %f", i, rsi[i])
		}
	}
	// All subsequent values should be 100 (no losses).
	for i := 14; i < len(rsi); i++ {
		if !approxEqual(rsi[i], 100, 0.01) {
			t.Errorf("RSI[%d] = %f, want ~100 in pure uptrend", i, rsi[i])
		}
	}
}

func TestRSI_Downtrend(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(30 - i)
	}
	candles := closesOnly(closes)
	rsi := RSI(candles, 14)

	for i := 14; i < len(rsi); i++ {
		if !approxEqual(rsi[i], 0, 0.01) {
			t.Errorf("RSI[%d] = %f, want ~0 in pure downtrend", i, rsi[i])
		}
	}
}

func TestRSI_Alternating(t *testing.T) {
	// Alternating up/down by same amount -> RSI should be near 50.
	closes := make([]float64, 40)
	closes[0] = 50
	for i := 1; i < len(closes); i++ {
		if i%2 == 1 {
			closes[i] = closes[i-1] + 1
		} else {
			closes[i] = closes[i-1] - 1
		}
	}
	candles := closesOnly(closes)
	rsi := RSI(candles, 14)

	// Check last few values are near 50.
	for i := len(rsi) - 5; i < len(rsi); i++ {
		if math.IsNaN(rsi[i]) || rsi[i] < 40 || rsi[i] > 60 {
			t.Errorf("RSI[%d] = %f, expected near 50 for alternating data", i, rsi[i])
		}
	}
}

// --- MACD Tests ---

func TestMACD_TrendChange(t *testing.T) {
	// Create an uptrend followed by a downtrend. MACD should cross signal.
	n := 80
	closes := make([]float64, n)
	for i := 0; i < 50; i++ {
		closes[i] = 100 + float64(i)*2 // uptrend
	}
	for i := 50; i < n; i++ {
		closes[i] = closes[49] - float64(i-49)*2 // downtrend
	}
	candles := closesOnly(closes)
	result := MACD(candles, 12, 26, 9)

	if len(result.MACD) != n {
		t.Fatalf("MACD length = %d, want %d", len(result.MACD), n)
	}

	// During uptrend, MACD line should be positive (fast EMA > slow EMA).
	// Check a value well into the uptrend.
	idx := 40
	if math.IsNaN(result.MACD[idx]) || result.MACD[idx] <= 0 {
		t.Errorf("MACD[%d] = %f, expected positive during uptrend", idx, result.MACD[idx])
	}

	// During downtrend, MACD should become negative.
	// Find if MACD goes negative at some point after reversal.
	foundNegative := false
	for i := 55; i < n; i++ {
		if !math.IsNaN(result.MACD[i]) && result.MACD[i] < 0 {
			foundNegative = true
			break
		}
	}
	if !foundNegative {
		t.Error("MACD never went negative during downtrend")
	}

	// Histogram should exist and be MACD - Signal.
	for i := 0; i < n; i++ {
		if !math.IsNaN(result.MACD[i]) && !math.IsNaN(result.Signal[i]) {
			expected := result.MACD[i] - result.Signal[i]
			if !approxEqual(result.Histogram[i], expected, 1e-9) {
				t.Errorf("Histogram[%d] = %f, want %f", i, result.Histogram[i], expected)
			}
		}
	}
}

// --- Bollinger Bands Tests ---

func TestBollingerBands_MiddleIsSMA(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = float64(100 + i)
	}
	candles := closesOnly(closes)
	bb := BollingerBands(candles, 20, 2.0)
	sma := SMA(candles, 20)

	for i := 0; i < len(closes); i++ {
		if !approxEqual(bb.Middle[i], sma[i], 1e-9) {
			t.Errorf("Middle[%d] = %f, SMA = %f", i, bb.Middle[i], sma[i])
		}
	}
}

func TestBollingerBands_Symmetry(t *testing.T) {
	closes := make([]float64, 25)
	for i := range closes {
		closes[i] = float64(50 + i%5)
	}
	candles := closesOnly(closes)
	bb := BollingerBands(candles, 20, 2.0)

	for i := 19; i < len(closes); i++ {
		upperDist := bb.Upper[i] - bb.Middle[i]
		lowerDist := bb.Middle[i] - bb.Lower[i]
		if !approxEqual(upperDist, lowerDist, 1e-9) {
			t.Errorf("Bands not symmetric at %d: upper dist=%f, lower dist=%f", i, upperDist, lowerDist)
		}
	}
}

func TestBollingerBands_VolatilityWidens(t *testing.T) {
	// Low volatility followed by high volatility.
	closes := make([]float64, 40)
	for i := 0; i < 20; i++ {
		closes[i] = 100 // flat
	}
	for i := 20; i < 40; i++ {
		if i%2 == 0 {
			closes[i] = 110
		} else {
			closes[i] = 90
		}
	}
	candles := closesOnly(closes)
	bb := BollingerBands(candles, 20, 2.0)

	// Bandwidth at index 19 (all flat) should be 0.
	bwFlat := bb.Upper[19] - bb.Lower[19]
	if !approxEqual(bwFlat, 0, 1e-9) {
		t.Errorf("Bandwidth during flat = %f, expected 0", bwFlat)
	}

	// Bandwidth at end should be wider than flat.
	bwVol := bb.Upper[39] - bb.Lower[39]
	if bwVol <= bwFlat+1e-9 {
		t.Errorf("Bandwidth during volatility (%f) should be > flat (%f)", bwVol, bwFlat)
	}
}

// --- Ichimoku Tests ---

func TestIchimoku_Basic(t *testing.T) {
	// Generate 60 candles with a known trend.
	n := 60
	data := make([][3]float64, n)
	for i := 0; i < n; i++ {
		base := float64(100 + i)
		data[i] = [3]float64{base + 2, base - 2, base} // high, low, close
	}
	candles := hlcCandles(data)
	result := Ichimoku(candles, 9, 26, 52)

	// Tenkan should be valid from index 8 onward.
	if math.IsNaN(result.Tenkan[8]) {
		t.Error("Tenkan[8] should not be NaN")
	}
	for i := 0; i < 8; i++ {
		if !math.IsNaN(result.Tenkan[i]) {
			t.Errorf("Tenkan[%d] should be NaN", i)
		}
	}

	// Kijun should be valid from index 25 onward.
	if math.IsNaN(result.Kijun[25]) {
		t.Error("Kijun[25] should not be NaN")
	}
	for i := 0; i < 25; i++ {
		if !math.IsNaN(result.Kijun[i]) {
			t.Errorf("Kijun[%d] should be NaN", i)
		}
	}

	// SenkouA should have values starting at index kijun + (kijun-1) = 51.
	if math.IsNaN(result.SenkouA[51]) {
		t.Error("SenkouA[51] should not be NaN")
	}

	// SenkouB should have values from index senkou-1 + kijun = 77.
	if math.IsNaN(result.SenkouB[77]) {
		t.Error("SenkouB[77] should not be NaN")
	}

	// Chikou: value at index 0 should equal candles[26].Close.
	if !approxEqual(result.Chikou[0], candles[26].Close, 1e-9) {
		t.Errorf("Chikou[0] = %f, want %f", result.Chikou[0], candles[26].Close)
	}
}

func TestIchimoku_TenkanKijunCrossover(t *testing.T) {
	// Create data where tenkan crosses above kijun (bullish signal).
	// Start flat, then trend up sharply -> tenkan (faster) should rise above kijun.
	n := 60
	data := make([][3]float64, n)
	for i := 0; i < 30; i++ {
		data[i] = [3]float64{102, 98, 100}
	}
	for i := 30; i < n; i++ {
		base := 100 + float64(i-30)*3
		data[i] = [3]float64{base + 2, base - 2, base}
	}
	candles := hlcCandles(data)
	result := Ichimoku(candles, 9, 26, 52)

	// After the trend starts, tenkan should eventually exceed kijun.
	foundCross := false
	for i := 35; i < n; i++ {
		if !math.IsNaN(result.Tenkan[i]) && !math.IsNaN(result.Kijun[i]) {
			if result.Tenkan[i] > result.Kijun[i] {
				foundCross = true
				break
			}
		}
	}
	if !foundCross {
		t.Error("Expected Tenkan to cross above Kijun during uptrend")
	}
}

// --- Edge Cases ---

func TestEmptyCandles(t *testing.T) {
	var candles []Candle
	sma := SMA(candles, 3)
	if len(sma) != 0 {
		t.Errorf("SMA of empty candles should be empty, got length %d", len(sma))
	}
	ema := EMA(candles, 3)
	if len(ema) != 0 {
		t.Errorf("EMA of empty candles should be empty, got length %d", len(ema))
	}
	rsi := RSI(candles, 14)
	if len(rsi) != 0 {
		t.Errorf("RSI of empty candles should be empty, got length %d", len(rsi))
	}
}
