package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/engine"
)

// BacktestConfig holds the parameters for a backtest run.
type BacktestConfig struct {
	Symbol    string
	Timeframe string
	From, To  time.Time
	Capital   float64
	MakerFee  float64
	TakerFee  float64
	Slippage  float64
	TradeSize float64 // fixed size per trade (units)
}

// BacktestResult holds the output of a completed backtest.
type BacktestResult struct {
	Symbol, Timeframe, Period     string
	Capital, FinalEquity          float64
	TotalPnL, TotalReturn        float64 // TotalReturn in percentage
	TotalTrades, WinCount, LossCount int
	WinRate, AvgWin, AvgLoss     float64
	ProfitFactor                 float64
	MaxDrawdown, SharpeRatio     float64
	Trades                       []TradeRecord
	EquityCurve                  []EquityPoint
}

// EquityPoint records portfolio equity at a specific time.
type EquityPoint struct {
	Time   time.Time
	Equity float64
	Price  float64
}

// Engine is the backtest time-loop orchestrator.
type Engine struct {
	Bus *engine.EventBus // optional (can be nil), for streaming progress
}

// Run executes a backtest over the given candles using the provided strategy.
func (e *Engine) Run(ctx context.Context, cfg BacktestConfig, candles []adapter.Candle, strat StrategyRunner) (*BacktestResult, error) {
	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles provided")
	}

	portfolio := NewPortfolio(cfg.Capital, cfg.MakerFee, cfg.TakerFee, cfg.Slippage)
	equityCurve := make([]EquityPoint, 0, len(candles))
	total := len(candles)

	for i, candle := range candles {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		currentPrice := candle.Close

		// Tick: check SL/TP
		portfolio.Tick(currentPrice, candle.Timestamp)

		// Feed candle window up to current index to strategy
		window := candles[:i+1]
		signal := strat.Evaluate(cfg.Symbol, window)

		// Process signal
		if signal != nil {
			switch signal.Action {
			case ActionBuy:
				// Only open if no existing position
				if len(portfolio.Positions) == 0 {
					err := portfolio.OpenPosition(cfg.Symbol, SideLong, cfg.TradeSize, currentPrice, candle.Timestamp)
					if err == nil && (signal.StopLoss > 0 || signal.TakeProfit > 0) {
						// Set SL/TP on the newly opened position
						idx := len(portfolio.Positions) - 1
						if signal.StopLoss > 0 {
							portfolio.Positions[idx].StopLoss = signal.StopLoss
						}
						if signal.TakeProfit > 0 {
							portfolio.Positions[idx].TakeProfit = signal.TakeProfit
						}
					}
				}
			case ActionSell:
				// Close all long positions
				for len(portfolio.Positions) > 0 {
					portfolio.ClosePosition(0, currentPrice, candle.Timestamp)
				}
			}
		}

		// Record equity point
		equity := portfolio.Equity(currentPrice)
		equityCurve = append(equityCurve, EquityPoint{
			Time:   candle.Timestamp,
			Equity: equity,
			Price:  currentPrice,
		})

		// Emit progress every 10 candles or on last candle
		if i%10 == 0 || i == total-1 {
			e.publishEvent("backtest.progress", map[string]any{
				"index":     i,
				"total":     total,
				"equity":    equity,
				"price":     currentPrice,
				"symbol":    cfg.Symbol,
				"positions": len(portfolio.Positions),
			})
		}
	}

	// Close remaining positions at last price
	lastPrice := candles[len(candles)-1].Close
	lastTime := candles[len(candles)-1].Timestamp
	for len(portfolio.Positions) > 0 {
		portfolio.ClosePosition(0, lastPrice, lastTime)
	}

	// Calculate metrics
	result := e.buildResult(cfg, portfolio, equityCurve)

	// Emit complete event
	e.publishEvent("backtest.complete", map[string]any{
		"symbol":       result.Symbol,
		"total_pnl":    result.TotalPnL,
		"total_return":  result.TotalReturn,
		"total_trades":  result.TotalTrades,
		"win_rate":     result.WinRate,
		"max_drawdown": result.MaxDrawdown,
		"sharpe_ratio": result.SharpeRatio,
	})

	return result, nil
}

// publishEvent emits an event via Bus if it is non-nil.
func (e *Engine) publishEvent(eventType string, data map[string]any) {
	if e.Bus == nil {
		return
	}
	e.Bus.Publish(engine.Event{
		Type: eventType,
		Data: data,
	})
}

// buildResult computes the final BacktestResult from portfolio state and equity curve.
func (e *Engine) buildResult(cfg BacktestConfig, portfolio *Portfolio, equityCurve []EquityPoint) *BacktestResult {
	trades := portfolio.Trades
	totalTrades := len(trades)

	var winCount, lossCount int
	var totalWin, totalLoss float64
	for _, tr := range trades {
		if tr.PnL > 0 {
			winCount++
			totalWin += tr.PnL
		} else {
			lossCount++
			totalLoss += math.Abs(tr.PnL)
		}
	}

	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winCount) / float64(totalTrades)
	}

	avgWin := 0.0
	if winCount > 0 {
		avgWin = totalWin / float64(winCount)
	}

	avgLoss := 0.0
	if lossCount > 0 {
		avgLoss = totalLoss / float64(lossCount)
	}

	profitFactor := 0.0
	if totalLoss > 0 {
		profitFactor = totalWin / totalLoss
	}

	// Final equity from last equity point (after closing remaining)
	finalEquity := portfolio.Cash
	totalPnL := finalEquity - cfg.Capital
	totalReturn := 0.0
	if cfg.Capital > 0 {
		totalReturn = (totalPnL / cfg.Capital) * 100.0
	}

	// Compute per-period returns for Sharpe
	equities := make([]float64, len(equityCurve))
	for i, ep := range equityCurve {
		equities[i] = ep.Equity
	}

	returns := make([]float64, 0, len(equities)-1)
	for i := 1; i < len(equities); i++ {
		if equities[i-1] > 0 {
			returns = append(returns, (equities[i]-equities[i-1])/equities[i-1])
		}
	}

	period := ""
	if len(equityCurve) > 0 {
		from := equityCurve[0].Time
		to := equityCurve[len(equityCurve)-1].Time
		period = fmt.Sprintf("%s to %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	}

	return &BacktestResult{
		Symbol:       cfg.Symbol,
		Timeframe:    cfg.Timeframe,
		Period:       period,
		Capital:      cfg.Capital,
		FinalEquity:  finalEquity,
		TotalPnL:     totalPnL,
		TotalReturn:  totalReturn,
		TotalTrades:  totalTrades,
		WinCount:     winCount,
		LossCount:    lossCount,
		WinRate:      winRate,
		AvgWin:       avgWin,
		AvgLoss:      avgLoss,
		ProfitFactor: profitFactor,
		MaxDrawdown:  computeMaxDrawdown(equities),
		SharpeRatio:  computeSharpeRatio(returns),
		Trades:       trades,
		EquityCurve:  equityCurve,
	}
}

// computeSharpeRatio calculates the Sharpe ratio as mean/stddev of returns.
func computeSharpeRatio(returns []float64) float64 {
	n := len(returns)
	if n == 0 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(n)

	var variance float64
	for _, r := range returns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(n)

	stddev := math.Sqrt(variance)
	if stddev == 0 {
		return 0
	}

	return mean / stddev
}

// computeMaxDrawdown calculates the maximum drawdown as a fraction of peak equity.
func computeMaxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0
	}

	peak := equity[0]
	maxDD := 0.0

	for _, e := range equity {
		if e > peak {
			peak = e
		}
		if peak > 0 {
			dd := (peak - e) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	return maxDD
}
