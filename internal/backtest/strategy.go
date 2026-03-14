package backtest

import (
	"math"
	"strconv"
	"strings"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/analysis"
	"github.com/clawtrade/clawtrade/internal/strategy"
)

// Action represents a trading action signal.
type Action string

const (
	ActionBuy  Action = "buy"
	ActionSell Action = "sell"
	ActionHold Action = "hold"
)

// BacktestSignal holds the result of a strategy evaluation.
type BacktestSignal struct {
	Action     Action
	Size       float64 // 0 = use default sizing
	StopLoss   float64
	TakeProfit float64
	Strength   float64 // 0-1
	Reason     string
}

// StrategyRunner is the interface all strategy modes implement.
type StrategyRunner interface {
	Evaluate(symbol string, candles []adapter.Candle) *BacktestSignal
}

// ---------------------------------------------------------------------------
// Mode 1: CodeStrategy — wraps existing strategy.Arena
// ---------------------------------------------------------------------------

// CodeStrategy wraps an existing strategy.Arena and extracts signals by name.
type CodeStrategy struct {
	Arena        *strategy.Arena
	StrategyName string
}

// Evaluate runs the arena strategies on close prices extracted from candles.
func (cs *CodeStrategy) Evaluate(symbol string, candles []adapter.Candle) *BacktestSignal {
	if len(candles) == 0 {
		return &BacktestSignal{Action: ActionHold}
	}

	prices := make([]float64, len(candles))
	for i, c := range candles {
		prices[i] = c.Close
	}

	signals := cs.Arena.RunSignal(symbol, prices)
	sig, ok := signals[cs.StrategyName]
	if !ok || sig == nil {
		return &BacktestSignal{Action: ActionHold}
	}

	action := ActionHold
	switch strings.ToLower(sig.Side) {
	case "buy":
		action = ActionBuy
	case "sell":
		action = ActionSell
	}

	return &BacktestSignal{
		Action:   action,
		Strength: sig.Strength,
		Reason:   sig.Reason,
	}
}

// ---------------------------------------------------------------------------
// Mode 2: ConfigStrategy — rule-based expression evaluation
// ---------------------------------------------------------------------------

// ConfigStrategy evaluates simple rule expressions against indicator values.
type ConfigStrategy struct {
	BuyWhen  string // e.g. "rsi < 30 AND close > sma_50"
	SellWhen string // e.g. "rsi > 70"
}

// Evaluate computes indicators on the candles and checks buy/sell rules.
func (cs *ConfigStrategy) Evaluate(symbol string, candles []adapter.Candle) *BacktestSignal {
	if len(candles) == 0 {
		return &BacktestSignal{Action: ActionHold}
	}

	aCandles := toAnalysisCandles(candles)
	vals := buildIndicatorMap(aCandles)

	if cs.BuyWhen != "" && evalExpr(cs.BuyWhen, vals) {
		return &BacktestSignal{Action: ActionBuy, Strength: 0.5, Reason: cs.BuyWhen}
	}
	if cs.SellWhen != "" && evalExpr(cs.SellWhen, vals) {
		return &BacktestSignal{Action: ActionSell, Strength: 0.5, Reason: cs.SellWhen}
	}

	return &BacktestSignal{Action: ActionHold}
}

// toAnalysisCandles converts adapter candles to analysis candles.
func toAnalysisCandles(candles []adapter.Candle) []analysis.Candle {
	out := make([]analysis.Candle, len(candles))
	for i, c := range candles {
		out[i] = analysis.Candle{
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			Timestamp: c.Timestamp.Unix(),
		}
	}
	return out
}

// lastValid returns the last non-NaN value from a slice, or NaN if none.
func lastValid(vals []float64) float64 {
	if len(vals) == 0 {
		return math.NaN()
	}
	return vals[len(vals)-1]
}

// buildIndicatorMap computes all available indicators and returns a map of
// their values at the last candle position.
func buildIndicatorMap(candles []analysis.Candle) map[string]float64 {
	n := len(candles)
	if n == 0 {
		return nil
	}

	vals := make(map[string]float64)

	// Price values at last candle
	last := candles[n-1]
	vals["close"] = last.Close
	vals["open"] = last.Open
	vals["high"] = last.High
	vals["low"] = last.Low
	vals["volume"] = last.Volume

	// RSI
	rsi := analysis.RSI(candles, 14)
	vals["rsi"] = lastValid(rsi)

	// SMA variants
	for _, p := range []int{10, 20, 50, 100, 200} {
		sma := analysis.SMA(candles, p)
		vals["sma_"+strconv.Itoa(p)] = lastValid(sma)
	}

	// EMA variants
	for _, p := range []int{9, 12, 21, 26, 50} {
		ema := analysis.EMA(candles, p)
		vals["ema_"+strconv.Itoa(p)] = lastValid(ema)
	}

	// MACD (12, 26, 9)
	macd := analysis.MACD(candles, 12, 26, 9)
	vals["macd"] = lastValid(macd.MACD)
	vals["macd_signal"] = lastValid(macd.Signal)
	vals["macd_hist"] = lastValid(macd.Histogram)

	// Bollinger Bands (20, 2.0)
	bb := analysis.BollingerBands(candles, 20, 2.0)
	vals["bb_upper"] = lastValid(bb.Upper)
	vals["bb_lower"] = lastValid(bb.Lower)

	return vals
}

// ---------------------------------------------------------------------------
// Expression evaluator
// ---------------------------------------------------------------------------

// evalExpr evaluates a compound expression joined by " AND " (case insensitive).
// All conditions must be true for the expression to be true.
func evalExpr(expr string, vals map[string]float64) bool {
	// Split by " AND " case-insensitively
	parts := splitAND(expr)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !evalCondition(part, vals) {
			return false
		}
	}
	return true
}

// splitAND splits an expression by " AND " in a case-insensitive manner.
func splitAND(expr string) []string {
	lower := strings.ToLower(expr)
	var parts []string
	for {
		idx := strings.Index(lower, " and ")
		if idx < 0 {
			parts = append(parts, strings.TrimSpace(expr))
			break
		}
		parts = append(parts, strings.TrimSpace(expr[:idx]))
		expr = expr[idx+5:]
		lower = lower[idx+5:]
	}
	return parts
}

// evalCondition evaluates a single condition: "name op value".
func evalCondition(cond string, vals map[string]float64) bool {
	// Try each operator (check longer ones first to avoid ambiguity)
	for _, op := range []string{"<=", ">=", "<", ">"} {
		idx := strings.Index(cond, op)
		if idx < 0 {
			continue
		}
		lhs := strings.TrimSpace(cond[:idx])
		rhs := strings.TrimSpace(cond[idx+len(op):])

		lVal := resolveValue(lhs, vals)
		rVal := resolveValue(rhs, vals)

		if math.IsNaN(lVal) || math.IsNaN(rVal) {
			return false
		}

		switch op {
		case "<":
			return lVal < rVal
		case ">":
			return lVal > rVal
		case "<=":
			return lVal <= rVal
		case ">=":
			return lVal >= rVal
		}
	}
	return false
}

// resolveValue looks up a name in the vals map or parses it as a float.
// Returns NaN if neither succeeds.
func resolveValue(s string, vals map[string]float64) float64 {
	s = strings.TrimSpace(s)
	if v, ok := vals[s]; ok {
		return v
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

// ---------------------------------------------------------------------------
// Mode 3: Built-in strategies
// ---------------------------------------------------------------------------

// EvalExprPublic evaluates an expression string against indicator values.
// Exported for use by the alerting system.
func EvalExprPublic(expr string, indicators map[string]float64) bool {
	return evalExpr(expr, indicators)
}

// ComputeIndicatorsPublic computes technical indicators from candles.
// Exported for use by the alerting system.
func ComputeIndicatorsPublic(candles []adapter.Candle) map[string]float64 {
	return buildIndicatorMap(toAnalysisCandles(candles))
}

// GetBuiltinStrategy returns a pre-configured ConfigStrategy for well-known
// strategy names.
func GetBuiltinStrategy(name string) StrategyRunner {
	switch strings.ToLower(name) {
	case "momentum":
		return &ConfigStrategy{BuyWhen: "rsi < 35 AND close > ema_21", SellWhen: "rsi > 70"}
	case "meanrevert", "mean_revert":
		return &ConfigStrategy{BuyWhen: "close < bb_lower", SellWhen: "close > bb_upper"}
	case "macd_cross", "macd":
		return &ConfigStrategy{BuyWhen: "macd > macd_signal AND macd_hist > 0", SellWhen: "macd < macd_signal AND macd_hist < 0"}
	default:
		return &ConfigStrategy{BuyWhen: "rsi < 30", SellWhen: "rsi > 70"}
	}
}
