package analysis

import (
	"math"
	"sync"
)

// ScreenerCondition defines a single filter criterion.
type ScreenerCondition struct {
	Indicator string  // "rsi", "macd", "sma", "ema", "price", "volume", "bollinger"
	Operator  string  // "above", "below", "crosses_above", "crosses_below", "between"
	Value     float64 // threshold value
	Value2    float64 // second threshold for "between" operator
	Period    int     // indicator period (e.g., RSI 14)
	Period2   int     // secondary period (e.g., SMA crossover)
}

// ScreenerResult holds a single symbol's screening result.
type ScreenerResult struct {
	Symbol     string
	Conditions []ConditionResult
	PassedAll  bool
	Score      float64 // 0-1, fraction of conditions met
}

// ConditionResult holds the evaluation of a single condition.
type ConditionResult struct {
	Condition ScreenerCondition
	Passed    bool
	Value     float64 // actual indicator value
}

// Screener scans multiple symbols against a set of conditions.
type Screener struct {
	Conditions []ScreenerCondition
}

// NewScreener creates a screener with the given conditions.
func NewScreener(conditions []ScreenerCondition) *Screener {
	return &Screener{Conditions: conditions}
}

// Screen evaluates a single symbol's candles against all conditions.
func (s *Screener) Screen(symbol string, candles []Candle) ScreenerResult {
	result := ScreenerResult{
		Symbol: symbol,
	}
	if len(s.Conditions) == 0 {
		result.PassedAll = true
		result.Score = 1.0
		return result
	}

	passed := 0
	for _, cond := range s.Conditions {
		cr := evaluateCondition(cond, candles)
		result.Conditions = append(result.Conditions, cr)
		if cr.Passed {
			passed++
		}
	}

	result.Score = float64(passed) / float64(len(s.Conditions))
	result.PassedAll = passed == len(s.Conditions)
	return result
}

// ScreenMultiple evaluates multiple symbols concurrently.
func (s *Screener) ScreenMultiple(data map[string][]Candle) []ScreenerResult {
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]ScreenerResult, 0, len(data))

	for sym, candles := range data {
		wg.Add(1)
		go func(symbol string, c []Candle) {
			defer wg.Done()
			r := s.Screen(symbol, c)
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(sym, candles)
	}
	wg.Wait()
	return results
}

// indicatorValues computes indicator series from candles, returning the slice
// of float64 values. Returns nil if the data is insufficient.
func indicatorValues(indicator string, period int, candles []Candle) []float64 {
	switch indicator {
	case "rsi":
		return RSI(candles, period)
	case "sma":
		return SMA(candles, period)
	case "ema":
		return EMA(candles, period)
	case "macd":
		r := MACD(candles, 12, 26, 9)
		return r.MACD
	case "bollinger":
		p := period
		if p <= 0 {
			p = 20
		}
		bb := BollingerBands(candles, p, 2.0)
		// Compute %B: (close - lower) / (upper - lower)
		n := len(candles)
		pctB := make([]float64, n)
		for i := 0; i < n; i++ {
			if math.IsNaN(bb.Upper[i]) || math.IsNaN(bb.Lower[i]) {
				pctB[i] = math.NaN()
				continue
			}
			bw := bb.Upper[i] - bb.Lower[i]
			if bw == 0 {
				pctB[i] = 0.5
			} else {
				pctB[i] = (candles[i].Close - bb.Lower[i]) / bw
			}
		}
		return pctB
	case "price":
		vals := make([]float64, len(candles))
		for i, c := range candles {
			vals[i] = c.Close
		}
		return vals
	case "volume":
		vals := make([]float64, len(candles))
		for i, c := range candles {
			vals[i] = c.Volume
		}
		return vals
	default:
		return nil
	}
}

// lastValid returns the last non-NaN value and its index. Returns NaN, -1 if none.
func lastValid(vals []float64) (float64, int) {
	for i := len(vals) - 1; i >= 0; i-- {
		if !math.IsNaN(vals[i]) {
			return vals[i], i
		}
	}
	return math.NaN(), -1
}

// evaluateCondition checks one condition against candle data.
func evaluateCondition(c ScreenerCondition, candles []Candle) ConditionResult {
	cr := ConditionResult{Condition: c}

	if len(candles) == 0 {
		cr.Value = math.NaN()
		return cr
	}

	vals := indicatorValues(c.Indicator, c.Period, candles)
	if vals == nil {
		cr.Value = math.NaN()
		return cr
	}

	current, idx := lastValid(vals)
	if idx < 0 {
		cr.Value = math.NaN()
		return cr
	}
	cr.Value = current

	switch c.Operator {
	case "above":
		cr.Passed = current > c.Value
	case "below":
		cr.Passed = current < c.Value
	case "between":
		cr.Passed = current >= c.Value && current <= c.Value2
	case "crosses_above":
		if idx < 1 {
			cr.Passed = false
			break
		}
		// Find previous valid value
		prev, prevIdx := lastValid(vals[:idx])
		if prevIdx < 0 {
			cr.Passed = false
			break
		}
		cr.Passed = prev <= c.Value && current > c.Value
	case "crosses_below":
		if idx < 1 {
			cr.Passed = false
			break
		}
		prev, prevIdx := lastValid(vals[:idx])
		if prevIdx < 0 {
			cr.Passed = false
			break
		}
		cr.Passed = prev >= c.Value && current < c.Value
	}

	return cr
}
