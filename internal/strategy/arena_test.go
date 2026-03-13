package strategy

import (
	"sync"
	"testing"
	"time"
)

// helper: simple bullish strategy that always returns a buy signal
func bullishStrategy(symbol string, prices []float64) *Signal {
	return &Signal{
		Symbol:    symbol,
		Side:      "buy",
		Strength:  0.8,
		Reason:    "always bullish",
		Timestamp: time.Now(),
	}
}

// helper: simple bearish strategy that always returns a sell signal
func bearishStrategy(symbol string, prices []float64) *Signal {
	return &Signal{
		Symbol:    symbol,
		Side:      "sell",
		Strength:  0.6,
		Reason:    "always bearish",
		Timestamp: time.Now(),
	}
}

// helper: strategy that returns nil (no signal)
func silentStrategy(symbol string, prices []float64) *Signal {
	return nil
}

func TestRegisterAddsStrategy(t *testing.T) {
	a := NewArena()
	err := a.Register("bull", "always buy", bullishStrategy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := a.ListStrategies()
	if len(list) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(list))
	}
	if list[0].Name != "bull" {
		t.Fatalf("expected name 'bull', got %q", list[0].Name)
	}
	if !list[0].Active {
		t.Fatal("expected strategy to be active by default")
	}
}

func TestRegisterDuplicateReturnsError(t *testing.T) {
	a := NewArena()
	_ = a.Register("bull", "v1", bullishStrategy)
	err := a.Register("bull", "v2", bullishStrategy)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestUnregisterRemovesStrategy(t *testing.T) {
	a := NewArena()
	_ = a.Register("bull", "desc", bullishStrategy)

	err := a.Unregister("bull")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := a.ListStrategies()
	if len(list) != 0 {
		t.Fatalf("expected 0 strategies after unregister, got %d", len(list))
	}
}

func TestUnregisterNotFoundReturnsError(t *testing.T) {
	a := NewArena()
	err := a.Unregister("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistering nonexistent strategy")
	}
}

func TestSetActiveTogglesStrategy(t *testing.T) {
	a := NewArena()
	_ = a.Register("bull", "desc", bullishStrategy)

	err := a.SetActive("bull", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := a.ListStrategies()
	if list[0].Active {
		t.Fatal("expected strategy to be inactive")
	}

	_ = a.SetActive("bull", true)
	list = a.ListStrategies()
	if !list[0].Active {
		t.Fatal("expected strategy to be active again")
	}
}

func TestSetActiveNotFoundReturnsError(t *testing.T) {
	a := NewArena()
	err := a.SetActive("nonexistent", true)
	if err == nil {
		t.Fatal("expected error for nonexistent strategy")
	}
}

func TestRunSignalActiveOnly(t *testing.T) {
	a := NewArena()
	_ = a.Register("bull", "always buy", bullishStrategy)
	_ = a.Register("bear", "always sell", bearishStrategy)
	_ = a.Register("silent", "no signal", silentStrategy)

	// Deactivate bear
	_ = a.SetActive("bear", false)

	signals := a.RunSignal("BTC/USD", []float64{100, 101, 102})

	// bull should produce a signal, bear should not (inactive), silent returns nil
	if _, ok := signals["bull"]; !ok {
		t.Fatal("expected signal from bull strategy")
	}
	if _, ok := signals["bear"]; ok {
		t.Fatal("did not expect signal from inactive bear strategy")
	}
	if _, ok := signals["silent"]; ok {
		t.Fatal("did not expect signal from silent strategy")
	}

	if signals["bull"].Side != "buy" {
		t.Fatalf("expected buy signal, got %q", signals["bull"].Side)
	}
}

func TestRecordResultUpdatesWinLoss(t *testing.T) {
	a := NewArena()
	_ = a.Register("bull", "desc", bullishStrategy)

	_ = a.RecordResult("bull", 100)  // win
	_ = a.RecordResult("bull", -50)  // loss
	_ = a.RecordResult("bull", 200)  // win

	stats, err := a.GetStats("bull")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.WinCount != 2 {
		t.Fatalf("expected 2 wins, got %d", stats.WinCount)
	}
	if stats.LossCount != 1 {
		t.Fatalf("expected 1 loss, got %d", stats.LossCount)
	}
}

func TestRecordResultUpdatesPnL(t *testing.T) {
	a := NewArena()
	_ = a.Register("bull", "desc", bullishStrategy)

	_ = a.RecordResult("bull", 100)
	_ = a.RecordResult("bull", -50)
	_ = a.RecordResult("bull", 200)

	stats, _ := a.GetStats("bull")

	expectedPnL := 250.0
	if stats.TotalPnL != expectedPnL {
		t.Fatalf("expected PnL %.2f, got %.2f", expectedPnL, stats.TotalPnL)
	}

	// Win rate: 2/3
	expectedWinRate := 2.0 / 3.0
	if stats.WinRate < expectedWinRate-0.01 || stats.WinRate > expectedWinRate+0.01 {
		t.Fatalf("expected win rate ~%.4f, got %.4f", expectedWinRate, stats.WinRate)
	}

	// Avg win: (100+200)/2 = 150
	if stats.AvgWin != 150.0 {
		t.Fatalf("expected avg win 150, got %.2f", stats.AvgWin)
	}

	// Avg loss: 50/1 = 50
	if stats.AvgLoss != 50.0 {
		t.Fatalf("expected avg loss 50, got %.2f", stats.AvgLoss)
	}

	// Profit factor: 300/50 = 6.0
	if stats.ProfitFactor != 6.0 {
		t.Fatalf("expected profit factor 6.0, got %.2f", stats.ProfitFactor)
	}
}

func TestRecordResultNotFoundReturnsError(t *testing.T) {
	a := NewArena()
	err := a.RecordResult("nonexistent", 100)
	if err == nil {
		t.Fatal("expected error for nonexistent strategy")
	}
}

func TestGetRankingByWinRate(t *testing.T) {
	a := NewArena()
	_ = a.Register("high_wr", "high win rate", bullishStrategy)
	_ = a.Register("low_wr", "low win rate", bearishStrategy)

	// high_wr: 3 wins, 0 losses => 100% WR
	_ = a.RecordResult("high_wr", 10)
	_ = a.RecordResult("high_wr", 20)
	_ = a.RecordResult("high_wr", 5)

	// low_wr: 1 win, 2 losses => 33% WR
	_ = a.RecordResult("low_wr", 10)
	_ = a.RecordResult("low_wr", -20)
	_ = a.RecordResult("low_wr", -5)

	ranking := a.GetRanking("win_rate")
	if len(ranking) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ranking))
	}
	if ranking[0].Name != "high_wr" {
		t.Fatalf("expected high_wr first, got %q", ranking[0].Name)
	}
	if ranking[1].Name != "low_wr" {
		t.Fatalf("expected low_wr second, got %q", ranking[1].Name)
	}
}

func TestGetRankingByPnL(t *testing.T) {
	a := NewArena()
	_ = a.Register("profitable", "desc", bullishStrategy)
	_ = a.Register("unprofitable", "desc", bearishStrategy)

	_ = a.RecordResult("profitable", 500)
	_ = a.RecordResult("unprofitable", -200)

	ranking := a.GetRanking("pnl")
	if ranking[0].Name != "profitable" {
		t.Fatalf("expected profitable first, got %q", ranking[0].Name)
	}
}

func TestCompareTwoStrategies(t *testing.T) {
	a := NewArena()
	_ = a.Register("alpha", "desc", bullishStrategy)
	_ = a.Register("beta", "desc", bearishStrategy)

	_ = a.RecordResult("alpha", 100)
	_ = a.RecordResult("alpha", 50)
	_ = a.RecordResult("beta", -30)
	_ = a.RecordResult("beta", 10)

	cmp, err := a.Compare("alpha", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmp.Winner != "alpha" {
		t.Fatalf("expected alpha to win, got %q", cmp.Winner)
	}
	if cmp.Stats1.TotalPnL != 150 {
		t.Fatalf("expected alpha PnL 150, got %.2f", cmp.Stats1.TotalPnL)
	}
	if cmp.Stats2.TotalPnL != -20 {
		t.Fatalf("expected beta PnL -20, got %.2f", cmp.Stats2.TotalPnL)
	}
	if cmp.Verdict == "" {
		t.Fatal("expected non-empty verdict")
	}
}

func TestCompareNotFoundReturnsError(t *testing.T) {
	a := NewArena()
	_ = a.Register("alpha", "desc", bullishStrategy)

	_, err := a.Compare("alpha", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent strategy")
	}
}

func TestGetStatsReturnsCorrectStats(t *testing.T) {
	a := NewArena()
	_ = a.Register("test", "desc", bullishStrategy)

	_ = a.RecordResult("test", 100)
	_ = a.RecordResult("test", -30)

	stats, err := a.GetStats("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalPnL != 70 {
		t.Fatalf("expected PnL 70, got %.2f", stats.TotalPnL)
	}
	if stats.WinCount != 1 {
		t.Fatalf("expected 1 win, got %d", stats.WinCount)
	}
	if stats.LossCount != 1 {
		t.Fatalf("expected 1 loss, got %d", stats.LossCount)
	}
	if stats.MaxDrawdown != 30 {
		t.Fatalf("expected max drawdown 30, got %.2f", stats.MaxDrawdown)
	}
}

func TestGetStatsNotFoundReturnsError(t *testing.T) {
	a := NewArena()
	_, err := a.GetStats("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent strategy")
	}
}

func TestListStrategiesReturnsAll(t *testing.T) {
	a := NewArena()
	_ = a.Register("a", "desc a", bullishStrategy)
	_ = a.Register("b", "desc b", bearishStrategy)
	_ = a.Register("c", "desc c", silentStrategy)

	list := a.ListStrategies()
	if len(list) != 3 {
		t.Fatalf("expected 3 strategies, got %d", len(list))
	}

	names := make(map[string]bool)
	for _, e := range list {
		names[e.Name] = true
	}
	for _, n := range []string{"a", "b", "c"} {
		if !names[n] {
			t.Fatalf("expected strategy %q in list", n)
		}
	}
}

func TestConcurrentRegisterAndRun(t *testing.T) {
	a := NewArena()
	var wg sync.WaitGroup

	// Concurrently register strategies
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "strategy_" + string(rune('A'+idx))
			_ = a.Register(name, "concurrent", func(symbol string, prices []float64) *Signal {
				return &Signal{
					Symbol:    symbol,
					Side:      "buy",
					Strength:  0.5,
					Reason:    "concurrent test",
					Timestamp: time.Now(),
				}
			})
		}(i)
	}

	// Concurrently run signals while registering
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = a.RunSignal("ETH/USD", []float64{50, 51, 52})
		}()
	}

	// Concurrently record results
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "strategy_" + string(rune('A'+idx))
			_ = a.RecordResult(name, float64(idx)*10)
		}(i)
	}

	wg.Wait()

	// Verify no panic and data is consistent
	list := a.ListStrategies()
	if len(list) == 0 {
		t.Fatal("expected at least some strategies to be registered")
	}
}
