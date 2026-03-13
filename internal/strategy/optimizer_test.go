package strategy

import (
	"math"
	"testing"
)

// Simple quadratic: -(x-3)^2 + 10, max at x=3, score=10
func quadratic(params map[string]float64) float64 {
	x := params["x"]
	return -(x-3)*(x-3) + 10
}

// Multi-param: -(x-2)^2 - (y-5)^2 + 20, max at x=2, y=5, score=20
func multiParamObjective(params map[string]float64) float64 {
	x := params["x"]
	y := params["y"]
	return -(x-2)*(x-2) - (y-5)*(y-5) + 20
}

func TestGridSearchFindsOptimalForQuadratic(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 6, Step: 1, Current: 0},
	}
	opt := NewOptimizer(params)
	result := opt.GridSearch(quadratic)

	if result.BestParams.Values["x"] != 3.0 {
		t.Errorf("expected x=3, got x=%v", result.BestParams.Values["x"])
	}
	if result.BestParams.Score != 10.0 {
		t.Errorf("expected score=10, got %v", result.BestParams.Score)
	}
	// 0,1,2,3,4,5,6 = 7 combinations
	if result.Iterations != 7 {
		t.Errorf("expected 7 iterations, got %d", result.Iterations)
	}
}

func TestGridSearchMultipleParameters(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 4, Step: 1, Current: 0},
		{Name: "y", Min: 3, Max: 7, Step: 1, Current: 3},
	}
	opt := NewOptimizer(params)
	result := opt.GridSearch(multiParamObjective)

	if result.BestParams.Values["x"] != 2.0 {
		t.Errorf("expected x=2, got x=%v", result.BestParams.Values["x"])
	}
	if result.BestParams.Values["y"] != 5.0 {
		t.Errorf("expected y=5, got y=%v", result.BestParams.Values["y"])
	}
	if result.BestParams.Score != 20.0 {
		t.Errorf("expected score=20, got %v", result.BestParams.Score)
	}
	// 5 x-values * 5 y-values = 25
	expectedCombinations := 25
	if len(result.AllResults) != expectedCombinations {
		t.Errorf("expected %d results, got %d", expectedCombinations, len(result.AllResults))
	}
}

func TestRandomSearchFindsReasonableParams(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 6, Step: 0.5, Current: 0},
	}
	opt := NewOptimizer(params)
	result := opt.RandomSearch(quadratic, 200)

	// With 200 iterations over a small range, should get close to optimal
	if result.BestParams.Score < 9.0 {
		t.Errorf("expected score >= 9.0, got %v (x=%v)",
			result.BestParams.Score, result.BestParams.Values["x"])
	}
	if result.Iterations != 200 {
		t.Errorf("expected 200 iterations, got %d", result.Iterations)
	}
}

func TestHillClimbImprovesFromStartingPoint(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 6, Step: 1, Current: 0},
	}
	opt := NewOptimizer(params)
	result := opt.HillClimb(quadratic, 20)

	// Starting at x=0: score = -9+10 = 1. Should improve.
	if result.BestParams.Score <= 1.0 {
		t.Errorf("expected improvement from initial score of 1, got %v", result.BestParams.Score)
	}
	if result.Improvement <= 0 {
		t.Errorf("expected positive improvement, got %v", result.Improvement)
	}
}

func TestHillClimbConvergesToLocalOptimum(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 6, Step: 1, Current: 0},
	}
	opt := NewOptimizer(params)
	result := opt.HillClimb(quadratic, 100)

	// Should converge to x=3
	if result.BestParams.Values["x"] != 3.0 {
		t.Errorf("expected convergence to x=3, got x=%v", result.BestParams.Values["x"])
	}
	if result.BestParams.Score != 10.0 {
		t.Errorf("expected score=10, got %v", result.BestParams.Score)
	}
}

func TestSnapToStep(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		min      float64
		max      float64
		step     float64
		expected float64
	}{
		{"exact step", 2.0, 0, 10, 1, 2.0},
		{"between steps rounds to nearest", 2.3, 0, 10, 1, 2.0},
		{"between steps rounds up", 2.7, 0, 10, 1, 3.0},
		{"at minimum", 0.0, 0, 10, 1, 0.0},
		{"at maximum", 10.0, 0, 10, 1, 10.0},
		{"below minimum clamps", -1.0, 0, 10, 1, 0.0},
		{"above maximum clamps", 11.0, 0, 10, 1, 10.0},
		{"fractional step", 0.35, 0, 1, 0.25, 0.25},
		{"zero step returns value clamped", 5.0, 0, 10, 0, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snapToStep(tt.value, tt.min, tt.max, tt.step)
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("snapToStep(%v, %v, %v, %v) = %v, want %v",
					tt.value, tt.min, tt.max, tt.step, got, tt.expected)
			}
		})
	}
}

func TestGenerateCombinationsCorrectCount(t *testing.T) {
	params := []Parameter{
		{Name: "a", Min: 0, Max: 2, Step: 1, Current: 0},   // 3 values: 0,1,2
		{Name: "b", Min: 0, Max: 4, Step: 2, Current: 0},   // 3 values: 0,2,4
		{Name: "c", Min: 10, Max: 12, Step: 1, Current: 10}, // 3 values: 10,11,12
	}
	combos := generateCombinations(params)

	expected := 3 * 3 * 3 // 27
	if len(combos) != expected {
		t.Errorf("expected %d combinations, got %d", expected, len(combos))
	}

	// Each combination should have all 3 parameter names
	for i, c := range combos {
		if len(c) != 3 {
			t.Errorf("combination %d has %d params, expected 3", i, len(c))
		}
		for _, p := range params {
			if _, ok := c[p.Name]; !ok {
				t.Errorf("combination %d missing param %s", i, p.Name)
			}
		}
	}
}

func TestGetBestReturnsAfterOptimization(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 6, Step: 1, Current: 0},
	}
	opt := NewOptimizer(params)

	// Before optimization, best should be nil
	if opt.GetBest() != nil {
		t.Error("expected nil before optimization")
	}

	opt.GridSearch(quadratic)
	best := opt.GetBest()

	if best == nil {
		t.Fatal("expected non-nil best after optimization")
	}
	if best.Values["x"] != 3.0 {
		t.Errorf("expected x=3, got x=%v", best.Values["x"])
	}
	if best.Score != 10.0 {
		t.Errorf("expected score=10, got %v", best.Score)
	}
}

func TestGetHistoryTracksAllEvaluations(t *testing.T) {
	params := []Parameter{
		{Name: "x", Min: 0, Max: 4, Step: 1, Current: 0},
	}
	opt := NewOptimizer(params)

	// Before optimization, history should be empty
	if len(opt.GetHistory()) != 0 {
		t.Error("expected empty history before optimization")
	}

	opt.GridSearch(quadratic) // 5 combinations: 0,1,2,3,4
	history := opt.GetHistory()

	if len(history) != 5 {
		t.Errorf("expected 5 history entries, got %d", len(history))
	}

	// Run another search, history should accumulate
	opt.RandomSearch(quadratic, 10)
	history = opt.GetHistory()

	if len(history) != 15 {
		t.Errorf("expected 15 history entries after second search, got %d", len(history))
	}
}

func TestOptimizerWithSingleParameter(t *testing.T) {
	params := []Parameter{
		{Name: "threshold", Min: 0.1, Max: 0.9, Step: 0.1, Current: 0.5},
	}
	opt := NewOptimizer(params)

	// Objective: score is highest at threshold=0.5
	objective := func(p map[string]float64) float64 {
		th := p["threshold"]
		return -(th-0.5)*(th-0.5) + 1.0
	}

	result := opt.GridSearch(objective)

	if math.Abs(result.BestParams.Values["threshold"]-0.5) > 1e-9 {
		t.Errorf("expected threshold=0.5, got %v", result.BestParams.Values["threshold"])
	}
	if math.Abs(result.BestParams.Score-1.0) > 1e-9 {
		t.Errorf("expected score=1.0, got %v", result.BestParams.Score)
	}

	// Verify hill climb also works with single param
	opt2 := NewOptimizer(params)
	result2 := opt2.HillClimb(objective, 50)
	if math.Abs(result2.BestParams.Values["threshold"]-0.5) > 1e-9 {
		t.Errorf("hill climb: expected threshold=0.5, got %v", result2.BestParams.Values["threshold"])
	}
}
