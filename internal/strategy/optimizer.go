package strategy

import (
	"math"
	"math/rand"
	"sync"
)

// Parameter defines a tunable parameter with range.
type Parameter struct {
	Name    string  `json:"name"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Step    float64 `json:"step"`    // step size for grid search
	Current float64 `json:"current"` // current/initial value
}

// ParameterSet is a collection of parameter values.
type ParameterSet struct {
	Values map[string]float64 `json:"values"`
	Score  float64            `json:"score"` // fitness score
}

// OptimizationResult holds the result of an optimization run.
type OptimizationResult struct {
	BestParams  ParameterSet   `json:"best_params"`
	AllResults  []ParameterSet `json:"all_results"`
	Iterations  int            `json:"iterations"`
	Improvement float64        `json:"improvement"` // % improvement over initial
}

// ObjectiveFunc evaluates a parameter set and returns a score (higher = better).
type ObjectiveFunc func(params map[string]float64) float64

// Optimizer handles parameter auto-tuning.
type Optimizer struct {
	mu         sync.Mutex
	parameters []Parameter
	history    []ParameterSet
	bestResult *ParameterSet
}

// NewOptimizer creates an optimizer with given parameters.
func NewOptimizer(params []Parameter) *Optimizer {
	p := make([]Parameter, len(params))
	copy(p, params)
	return &Optimizer{
		parameters: p,
		history:    make([]ParameterSet, 0),
	}
}

// GridSearch exhaustively tests all parameter combinations.
func (o *Optimizer) GridSearch(objective ObjectiveFunc) OptimizationResult {
	combinations := generateCombinations(o.parameters)

	// Evaluate initial params for improvement calculation
	initialParams := make(map[string]float64, len(o.parameters))
	for _, p := range o.parameters {
		initialParams[p.Name] = p.Current
	}
	initialScore := objective(initialParams)

	var best ParameterSet
	bestInitialized := false
	allResults := make([]ParameterSet, 0, len(combinations))

	for _, combo := range combinations {
		score := objective(combo)
		ps := ParameterSet{
			Values: combo,
			Score:  score,
		}
		allResults = append(allResults, ps)

		if !bestInitialized || score > best.Score {
			best = ps
			bestInitialized = true
		}
	}

	o.mu.Lock()
	o.history = append(o.history, allResults...)
	if o.bestResult == nil || best.Score > o.bestResult.Score {
		cp := ParameterSet{Values: copyMap(best.Values), Score: best.Score}
		o.bestResult = &cp
	}
	o.mu.Unlock()

	improvement := 0.0
	if initialScore != 0 {
		improvement = ((best.Score - initialScore) / math.Abs(initialScore)) * 100
	} else if best.Score > 0 {
		improvement = 100.0
	}

	return OptimizationResult{
		BestParams:  best,
		AllResults:  allResults,
		Iterations:  len(combinations),
		Improvement: improvement,
	}
}

// RandomSearch samples random parameter combinations.
func (o *Optimizer) RandomSearch(objective ObjectiveFunc, iterations int) OptimizationResult {
	// Evaluate initial params for improvement calculation
	initialParams := make(map[string]float64, len(o.parameters))
	for _, p := range o.parameters {
		initialParams[p.Name] = p.Current
	}
	initialScore := objective(initialParams)

	var best ParameterSet
	bestInitialized := false
	allResults := make([]ParameterSet, 0, iterations)

	for i := 0; i < iterations; i++ {
		combo := make(map[string]float64, len(o.parameters))
		for _, p := range o.parameters {
			raw := p.Min + rand.Float64()*(p.Max-p.Min)
			combo[p.Name] = snapToStep(raw, p.Min, p.Max, p.Step)
		}

		score := objective(combo)
		ps := ParameterSet{
			Values: combo,
			Score:  score,
		}
		allResults = append(allResults, ps)

		if !bestInitialized || score > best.Score {
			best = ps
			bestInitialized = true
		}
	}

	o.mu.Lock()
	o.history = append(o.history, allResults...)
	if o.bestResult == nil || best.Score > o.bestResult.Score {
		cp := ParameterSet{Values: copyMap(best.Values), Score: best.Score}
		o.bestResult = &cp
	}
	o.mu.Unlock()

	improvement := 0.0
	if initialScore != 0 {
		improvement = ((best.Score - initialScore) / math.Abs(initialScore)) * 100
	} else if best.Score > 0 {
		improvement = 100.0
	}

	return OptimizationResult{
		BestParams:  best,
		AllResults:  allResults,
		Iterations:  iterations,
		Improvement: improvement,
	}
}

// HillClimb starts from current values and iteratively improves.
func (o *Optimizer) HillClimb(objective ObjectiveFunc, maxIterations int) OptimizationResult {
	// Start from current parameter values
	current := make(map[string]float64, len(o.parameters))
	for _, p := range o.parameters {
		current[p.Name] = snapToStep(p.Current, p.Min, p.Max, p.Step)
	}

	initialScore := objective(current)
	currentScore := initialScore

	allResults := make([]ParameterSet, 0, maxIterations)
	allResults = append(allResults, ParameterSet{Values: copyMap(current), Score: currentScore})

	iterations := 0
	for i := 0; i < maxIterations; i++ {
		iterations++
		improved := false

		for _, p := range o.parameters {
			// Try +step
			candidate := copyMap(current)
			upVal := snapToStep(current[p.Name]+p.Step, p.Min, p.Max, p.Step)
			candidate[p.Name] = upVal
			upScore := objective(candidate)
			allResults = append(allResults, ParameterSet{Values: copyMap(candidate), Score: upScore})

			if upScore > currentScore {
				current[p.Name] = upVal
				currentScore = upScore
				improved = true
				continue
			}

			// Try -step
			candidate = copyMap(current)
			downVal := snapToStep(current[p.Name]-p.Step, p.Min, p.Max, p.Step)
			candidate[p.Name] = downVal
			downScore := objective(candidate)
			allResults = append(allResults, ParameterSet{Values: copyMap(candidate), Score: downScore})

			if downScore > currentScore {
				current[p.Name] = downVal
				currentScore = downScore
				improved = true
			}
		}

		if !improved {
			break
		}
	}

	best := ParameterSet{Values: copyMap(current), Score: currentScore}

	o.mu.Lock()
	o.history = append(o.history, allResults...)
	if o.bestResult == nil || best.Score > o.bestResult.Score {
		cp := ParameterSet{Values: copyMap(best.Values), Score: best.Score}
		o.bestResult = &cp
	}
	o.mu.Unlock()

	improvement := 0.0
	if initialScore != 0 {
		improvement = ((currentScore - initialScore) / math.Abs(initialScore)) * 100
	} else if currentScore > 0 {
		improvement = 100.0
	}

	return OptimizationResult{
		BestParams:  best,
		AllResults:  allResults,
		Iterations:  iterations,
		Improvement: improvement,
	}
}

// GetBest returns the best parameter set found so far.
func (o *Optimizer) GetBest() *ParameterSet {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.bestResult == nil {
		return nil
	}
	cp := ParameterSet{Values: copyMap(o.bestResult.Values), Score: o.bestResult.Score}
	return &cp
}

// GetHistory returns all evaluated parameter sets.
func (o *Optimizer) GetHistory() []ParameterSet {
	o.mu.Lock()
	defer o.mu.Unlock()
	result := make([]ParameterSet, len(o.history))
	copy(result, o.history)
	return result
}

// snapToStep rounds a value to the nearest step within range.
func snapToStep(value, min, max, step float64) float64 {
	if step <= 0 {
		if value < min {
			return min
		}
		if value > max {
			return max
		}
		return value
	}

	// Snap to nearest step from min
	steps := math.Round((value - min) / step)
	snapped := min + steps*step

	// Clamp to range
	if snapped < min {
		snapped = min
	}
	if snapped > max {
		snapped = max
	}

	return snapped
}

// generateCombinations generates all grid search combinations.
func generateCombinations(params []Parameter) []map[string]float64 {
	if len(params) == 0 {
		return []map[string]float64{{}}
	}

	// Generate values for each parameter
	paramValues := make([][]float64, len(params))
	for i, p := range params {
		var values []float64
		if p.Step <= 0 {
			values = append(values, p.Current)
		} else {
			for v := p.Min; v <= p.Max+p.Step*0.01; v += p.Step {
				snapped := snapToStep(v, p.Min, p.Max, p.Step)
				if snapped <= p.Max {
					values = append(values, snapped)
				}
			}
			if len(values) == 0 {
				values = append(values, p.Min)
			}
		}
		paramValues[i] = values
	}

	// Calculate total combinations
	total := 1
	for _, vals := range paramValues {
		total *= len(vals)
	}

	combinations := make([]map[string]float64, 0, total)

	// Generate combinations using indices
	indices := make([]int, len(params))
	for {
		combo := make(map[string]float64, len(params))
		for i, p := range params {
			combo[p.Name] = paramValues[i][indices[i]]
		}
		combinations = append(combinations, combo)

		// Increment indices (odometer-style)
		carry := true
		for i := len(indices) - 1; i >= 0 && carry; i-- {
			indices[i]++
			if indices[i] < len(paramValues[i]) {
				carry = false
			} else {
				indices[i] = 0
			}
		}
		if carry {
			break
		}
	}

	return combinations
}

func copyMap(m map[string]float64) map[string]float64 {
	cp := make(map[string]float64, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
