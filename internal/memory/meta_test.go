package memory

import (
	"database/sql"
	"math"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMetaMemory_RecordAccess(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// First access creates the record.
	if err := mm.RecordAccess("mem-1", "rule"); err != nil {
		t.Fatalf("RecordAccess: %v", err)
	}

	top, err := mm.GetTopMemories(10)
	if err != nil {
		t.Fatalf("GetTopMemories: %v", err)
	}
	if len(top) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(top))
	}
	if top[0].MemoryID != "mem-1" {
		t.Errorf("expected memory_id=mem-1, got %s", top[0].MemoryID)
	}
	if top[0].MemoryType != "rule" {
		t.Errorf("expected memory_type=rule, got %s", top[0].MemoryType)
	}
	if top[0].AccessCount != 1 {
		t.Errorf("expected access_count=1, got %d", top[0].AccessCount)
	}

	// Second access increments count.
	if err := mm.RecordAccess("mem-1", "rule"); err != nil {
		t.Fatalf("RecordAccess: %v", err)
	}
	top, _ = mm.GetTopMemories(10)
	if top[0].AccessCount != 2 {
		t.Errorf("expected access_count=2, got %d", top[0].AccessCount)
	}
}

func TestMetaMemory_RecordOutcome(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// Record some accesses and outcomes.
	mm.RecordAccess("mem-1", "episode")
	mm.RecordOutcome("mem-1", true)
	mm.RecordOutcome("mem-1", true)
	mm.RecordOutcome("mem-1", false)

	top, err := mm.GetTopMemories(10)
	if err != nil {
		t.Fatalf("GetTopMemories: %v", err)
	}
	if len(top) != 1 {
		t.Fatalf("expected 1, got %d", len(top))
	}
	m := top[0]
	if m.PositiveCount != 2 {
		t.Errorf("expected 2 positive, got %d", m.PositiveCount)
	}
	if m.NegativeCount != 1 {
		t.Errorf("expected 1 negative, got %d", m.NegativeCount)
	}
	// Effectiveness should be roughly 2/3 * recency * usage (recency ~ 1.0 since just created)
	if m.Effectiveness < 0.5 || m.Effectiveness > 0.9 {
		t.Errorf("effectiveness %f out of expected range [0.5, 0.9]", m.Effectiveness)
	}
}

func TestMetaMemory_GetEffectiveness(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// Unknown memory returns 0.
	score, err := mm.GetEffectiveness("unknown")
	if err != nil {
		t.Fatalf("GetEffectiveness: %v", err)
	}
	if score != 0 {
		t.Errorf("expected 0 for unknown, got %f", score)
	}

	// Memory with outcomes.
	mm.RecordAccess("mem-2", "rule")
	mm.RecordOutcome("mem-2", true)

	score, err = mm.GetEffectiveness("mem-2")
	if err != nil {
		t.Fatalf("GetEffectiveness: %v", err)
	}
	if score < 0.8 {
		t.Errorf("expected high effectiveness, got %f", score)
	}
}

func TestMetaMemory_GetTopMemories_Ranking(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// Create a good memory.
	mm.RecordAccess("good", "rule")
	for i := 0; i < 5; i++ {
		mm.RecordOutcome("good", true)
	}

	// Create a bad memory.
	mm.RecordAccess("bad", "rule")
	for i := 0; i < 5; i++ {
		mm.RecordOutcome("bad", false)
	}

	// Create a mediocre memory.
	mm.RecordAccess("mid", "episode")
	mm.RecordOutcome("mid", true)
	mm.RecordOutcome("mid", false)

	top, err := mm.GetTopMemories(10)
	if err != nil {
		t.Fatalf("GetTopMemories: %v", err)
	}
	if len(top) != 3 {
		t.Fatalf("expected 3, got %d", len(top))
	}
	if top[0].MemoryID != "good" {
		t.Errorf("expected good first, got %s", top[0].MemoryID)
	}
	if top[len(top)-1].MemoryID != "bad" {
		t.Errorf("expected bad last, got %s", top[len(top)-1].MemoryID)
	}
}

func TestMetaMemory_GetTopMemories_Limit(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	for i := 0; i < 5; i++ {
		id := "mem-" + string(rune('A'+i))
		mm.RecordAccess(id, "rule")
	}

	top, err := mm.GetTopMemories(2)
	if err != nil {
		t.Fatalf("GetTopMemories: %v", err)
	}
	if len(top) != 2 {
		t.Errorf("expected 2, got %d", len(top))
	}
}

func TestMetaMemory_Prune(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// Good memory.
	mm.RecordAccess("good", "rule")
	for i := 0; i < 5; i++ {
		mm.RecordOutcome("good", true)
	}

	// Bad memory - all negative outcomes.
	mm.RecordAccess("bad", "rule")
	for i := 0; i < 5; i++ {
		mm.RecordOutcome("bad", false)
	}

	pruned, err := mm.Prune(0.3)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// "bad" should be below 0.3 (effectiveness = 0), "good" should not.
	foundBad := false
	for _, m := range pruned {
		if m.MemoryID == "bad" {
			foundBad = true
		}
		if m.MemoryID == "good" {
			t.Error("good memory should not be pruned")
		}
	}
	if !foundBad {
		t.Error("bad memory should be in prune list")
	}
}

func TestMetaMemory_AutoPromoteDemote(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// High-performing memory.
	mm.RecordAccess("star", "rule")
	for i := 0; i < 10; i++ {
		mm.RecordOutcome("star", true)
	}

	// Low-performing memory.
	mm.RecordAccess("dud", "rule")
	for i := 0; i < 10; i++ {
		mm.RecordOutcome("dud", false)
	}

	// Middle memory.
	mm.RecordAccess("mid", "episode")
	mm.RecordOutcome("mid", true)
	mm.RecordOutcome("mid", false)

	promoted, demoted, err := mm.AutoPromoteDemote(0.7, 0.3)
	if err != nil {
		t.Fatalf("AutoPromoteDemote: %v", err)
	}

	hasPromoted := false
	for _, id := range promoted {
		if id == "star" {
			hasPromoted = true
		}
	}
	if !hasPromoted {
		t.Errorf("star should be promoted, promoted=%v", promoted)
	}

	hasDemoted := false
	for _, id := range demoted {
		if id == "dud" {
			hasDemoted = true
		}
	}
	if !hasDemoted {
		t.Errorf("dud should be demoted, demoted=%v", demoted)
	}
}

func TestComputeEffectiveness(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		pos, neg int
		access   int
		wantMin  float64
		wantMax  float64
	}{
		{"no outcomes", 0, 0, 0, 0.5, 0.5},
		{"all positive", 10, 0, 5, 0.9, 1.01},
		{"all negative", 0, 10, 5, -0.01, 0.01},
		{"half and half", 5, 5, 5, 0.4, 0.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeEffectiveness(tt.pos, tt.neg, tt.access, now)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("computeEffectiveness(%d, %d, %d) = %f, want [%f, %f]",
					tt.pos, tt.neg, tt.access, score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestComputeEffectiveness_RecencyDecay(t *testing.T) {
	// Memory accessed now vs 60 days ago.
	now := time.Now()
	old := now.Add(-60 * 24 * time.Hour)

	recentScore := computeEffectiveness(5, 0, 3, now)
	oldScore := computeEffectiveness(5, 0, 3, old)

	if oldScore >= recentScore {
		t.Errorf("old score %f should be less than recent score %f", oldScore, recentScore)
	}
	// After 60 days with half-life ~30 days, score should be roughly 1/4.
	ratio := oldScore / recentScore
	if math.Abs(ratio-0.25) > 0.15 {
		t.Errorf("expected ratio ~0.25, got %f", ratio)
	}
}

func TestMetaMemory_RecordOutcome_CreatesRow(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	// RecordOutcome on a memory that was never accessed should still work.
	if err := mm.RecordOutcome("new-mem", true); err != nil {
		t.Fatalf("RecordOutcome on new memory: %v", err)
	}

	score, err := mm.GetEffectiveness("new-mem")
	if err != nil {
		t.Fatalf("GetEffectiveness: %v", err)
	}
	if score < 0.8 {
		t.Errorf("expected high score for all-positive, got %f", score)
	}
}

func TestMetaMemory_Concurrent(t *testing.T) {
	db := openTestDB(t)
	mm, err := NewMetaMemory(db)
	if err != nil {
		t.Fatalf("NewMetaMemory: %v", err)
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			mm.RecordAccess("concurrent", "test")
			mm.RecordOutcome("concurrent", true)
			mm.GetEffectiveness("concurrent")
			mm.GetTopMemories(5)
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	top, err := mm.GetTopMemories(10)
	if err != nil {
		t.Fatalf("GetTopMemories: %v", err)
	}
	if len(top) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(top))
	}
	if top[0].AccessCount != 10 {
		t.Errorf("expected 10 accesses, got %d", top[0].AccessCount)
	}
}
