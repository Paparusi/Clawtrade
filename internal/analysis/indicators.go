package analysis

import "math"

// Candle represents a single OHLCV candlestick.
type Candle struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp int64
}

// SMA calculates Simple Moving Average over close prices.
// The first period-1 values are NaN.
func SMA(candles []Candle, period int) []float64 {
	n := len(candles)
	out := make([]float64, n)
	if period <= 0 || n == 0 {
		return out
	}
	for i := 0; i < period-1 && i < n; i++ {
		out[i] = math.NaN()
	}
	// running sum
	var sum float64
	for i := 0; i < period && i < n; i++ {
		sum += candles[i].Close
	}
	if period <= n {
		out[period-1] = sum / float64(period)
	}
	for i := period; i < n; i++ {
		sum += candles[i].Close - candles[i-period].Close
		out[i] = sum / float64(period)
	}
	return out
}

// EMA calculates Exponential Moving Average over close prices.
// The first period-1 values are NaN, the value at index period-1 equals the SMA.
func EMA(candles []Candle, period int) []float64 {
	n := len(candles)
	out := make([]float64, n)
	if period <= 0 || n < period {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	for i := 0; i < period-1; i++ {
		out[i] = math.NaN()
	}
	// seed with SMA
	var sum float64
	for i := 0; i < period; i++ {
		sum += candles[i].Close
	}
	out[period-1] = sum / float64(period)

	k := 2.0 / float64(period+1)
	for i := period; i < n; i++ {
		out[i] = candles[i].Close*k + out[i-1]*(1-k)
	}
	return out
}

// emaFromValues computes EMA on a raw float64 slice (used internally).
func emaFromValues(values []float64, period int) []float64 {
	n := len(values)
	out := make([]float64, n)
	if period <= 0 || n < period {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	for i := 0; i < period-1; i++ {
		out[i] = math.NaN()
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	out[period-1] = sum / float64(period)

	k := 2.0 / float64(period+1)
	for i := period; i < n; i++ {
		out[i] = values[i]*k + out[i-1]*(1-k)
	}
	return out
}

// RSI calculates the Relative Strength Index over the given period.
// Returns a slice of RSI values (0-100). The first period values are NaN.
func RSI(candles []Candle, period int) []float64 {
	n := len(candles)
	out := make([]float64, n)
	if period <= 0 || n <= period {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}

	// First period values are NaN.
	for i := 0; i <= period-1; i++ {
		out[i] = math.NaN()
	}

	// Calculate initial average gain/loss using first period changes.
	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		change := candles[i].Close - candles[i-1].Close
		if change > 0 {
			avgGain += change
		} else {
			avgLoss -= change // make positive
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	if avgLoss == 0 {
		out[period] = 100
	} else {
		rs := avgGain / avgLoss
		out[period] = 100 - 100/(1+rs)
	}

	// Smoothed RSI for remaining values.
	for i := period + 1; i < n; i++ {
		change := candles[i].Close - candles[i-1].Close
		var gain, loss float64
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		if avgLoss == 0 {
			out[i] = 100
		} else {
			rs := avgGain / avgLoss
			out[i] = 100 - 100/(1+rs)
		}
	}
	return out
}

// MACDResult holds MACD calculation results.
type MACDResult struct {
	MACD      []float64 // MACD line (fast EMA - slow EMA)
	Signal    []float64 // Signal line (EMA of MACD)
	Histogram []float64 // MACD - Signal
}

// MACD calculates Moving Average Convergence Divergence.
// Standard parameters: fast=12, slow=26, signal=9.
func MACD(candles []Candle, fast, slow, signal int) MACDResult {
	n := len(candles)
	result := MACDResult{
		MACD:      make([]float64, n),
		Signal:    make([]float64, n),
		Histogram: make([]float64, n),
	}
	if n == 0 {
		return result
	}

	fastEMA := EMA(candles, fast)
	slowEMA := EMA(candles, slow)

	// MACD line = fastEMA - slowEMA
	macdLine := make([]float64, n)
	for i := 0; i < n; i++ {
		if math.IsNaN(fastEMA[i]) || math.IsNaN(slowEMA[i]) {
			macdLine[i] = math.NaN()
		} else {
			macdLine[i] = fastEMA[i] - slowEMA[i]
		}
	}
	result.MACD = macdLine

	// Find start of valid MACD values for signal computation.
	validStart := -1
	for i := 0; i < n; i++ {
		if !math.IsNaN(macdLine[i]) {
			validStart = i
			break
		}
	}
	if validStart < 0 || n-validStart < signal {
		// Not enough data for signal line.
		for i := range result.Signal {
			result.Signal[i] = math.NaN()
			result.Histogram[i] = math.NaN()
		}
		return result
	}

	// Compute signal as EMA of the valid MACD portion.
	validMACD := macdLine[validStart:]
	sigEMA := emaFromValues(validMACD, signal)

	for i := 0; i < validStart; i++ {
		result.Signal[i] = math.NaN()
		result.Histogram[i] = math.NaN()
	}
	for i := 0; i < len(sigEMA); i++ {
		result.Signal[validStart+i] = sigEMA[i]
		if math.IsNaN(sigEMA[i]) || math.IsNaN(macdLine[validStart+i]) {
			result.Histogram[validStart+i] = math.NaN()
		} else {
			result.Histogram[validStart+i] = macdLine[validStart+i] - sigEMA[i]
		}
	}
	return result
}

// BollingerResult holds Bollinger Bands calculation results.
type BollingerResult struct {
	Upper  []float64
	Middle []float64 // SMA
	Lower  []float64
}

// BollingerBands calculates Bollinger Bands.
// Standard: period=20, stdDev=2.0.
func BollingerBands(candles []Candle, period int, stdDev float64) BollingerResult {
	n := len(candles)
	result := BollingerResult{
		Upper:  make([]float64, n),
		Middle: make([]float64, n),
		Lower:  make([]float64, n),
	}

	sma := SMA(candles, period)
	result.Middle = sma

	for i := 0; i < n; i++ {
		if math.IsNaN(sma[i]) {
			result.Upper[i] = math.NaN()
			result.Lower[i] = math.NaN()
			continue
		}
		// Calculate standard deviation over the window.
		var sumSq float64
		for j := i - period + 1; j <= i; j++ {
			diff := candles[j].Close - sma[i]
			sumSq += diff * diff
		}
		sd := math.Sqrt(sumSq / float64(period))
		result.Upper[i] = sma[i] + stdDev*sd
		result.Lower[i] = sma[i] - stdDev*sd
	}
	return result
}

// IchimokuResult holds Ichimoku Cloud calculation results.
type IchimokuResult struct {
	Tenkan  []float64 // Conversion Line
	Kijun   []float64 // Base Line
	SenkouA []float64 // Leading Span A (shifted forward by kijun periods)
	SenkouB []float64 // Leading Span B (shifted forward by kijun periods)
	Chikou  []float64 // Lagging Span (shifted back by kijun periods)
}

// highLow returns the highest high and lowest low over [start, end) of candles.
func highLow(candles []Candle, start, end int) (float64, float64) {
	hi := candles[start].High
	lo := candles[start].Low
	for i := start + 1; i < end; i++ {
		if candles[i].High > hi {
			hi = candles[i].High
		}
		if candles[i].Low < lo {
			lo = candles[i].Low
		}
	}
	return hi, lo
}

// Ichimoku calculates the Ichimoku Cloud.
// Standard: tenkan=9, kijun=26, senkou=52.
// SenkouA and SenkouB are shifted forward by kijun periods (the output slices
// are extended by kijun elements). Chikou is shifted back by kijun periods.
func Ichimoku(candles []Candle, tenkan, kijun, senkou int) IchimokuResult {
	n := len(candles)
	outLen := n + kijun // extended for senkou forward shift

	result := IchimokuResult{
		Tenkan:  make([]float64, n),
		Kijun:   make([]float64, n),
		SenkouA: make([]float64, outLen),
		SenkouB: make([]float64, outLen),
		Chikou:  make([]float64, n),
	}

	// Initialize all to NaN.
	for i := 0; i < n; i++ {
		result.Tenkan[i] = math.NaN()
		result.Kijun[i] = math.NaN()
		result.Chikou[i] = math.NaN()
	}
	for i := 0; i < outLen; i++ {
		result.SenkouA[i] = math.NaN()
		result.SenkouB[i] = math.NaN()
	}

	// Tenkan-sen (conversion line): (highest high + lowest low) / 2 over tenkan period.
	for i := tenkan - 1; i < n; i++ {
		hi, lo := highLow(candles, i-tenkan+1, i+1)
		result.Tenkan[i] = (hi + lo) / 2
	}

	// Kijun-sen (base line): same calc over kijun period.
	for i := kijun - 1; i < n; i++ {
		hi, lo := highLow(candles, i-kijun+1, i+1)
		result.Kijun[i] = (hi + lo) / 2
	}

	// Senkou Span A: (Tenkan + Kijun) / 2, shifted forward by kijun periods.
	for i := kijun - 1; i < n; i++ {
		if !math.IsNaN(result.Tenkan[i]) && !math.IsNaN(result.Kijun[i]) {
			result.SenkouA[i+kijun] = (result.Tenkan[i] + result.Kijun[i]) / 2
		}
	}

	// Senkou Span B: (highest high + lowest low) / 2 over senkou period, shifted forward by kijun.
	for i := senkou - 1; i < n; i++ {
		hi, lo := highLow(candles, i-senkou+1, i+1)
		result.SenkouB[i+kijun] = (hi + lo) / 2
	}

	// Chikou Span: current close shifted back by kijun periods.
	for i := kijun; i < n; i++ {
		result.Chikou[i-kijun] = candles[i].Close
	}

	return result
}
