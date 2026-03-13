package memory

import (
	"database/sql"
	"math"
	"sync"
	"time"
)

// MemoryMeta holds metadata about a memory's effectiveness.
type MemoryMeta struct {
	MemoryID       string    `json:"memory_id"`
	MemoryType     string    `json:"memory_type"`
	AccessCount    int       `json:"access_count"`
	PositiveCount  int       `json:"positive_count"`
	NegativeCount  int       `json:"negative_count"`
	Effectiveness  float64   `json:"effectiveness"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// MetaMemory tracks which memories are useful and auto-promotes/demotes them.
type MetaMemory struct {
	mu sync.Mutex
	db *sql.DB
}

// NewMetaMemory creates a MetaMemory backed by the given database connection.
// It creates the required table if it does not exist.
func NewMetaMemory(db *sql.DB) (*MetaMemory, error) {
	mm := &MetaMemory{db: db}
	if err := mm.ensureTable(); err != nil {
		return nil, err
	}
	return mm, nil
}

func (mm *MetaMemory) ensureTable() error {
	_, err := mm.db.Exec(`CREATE TABLE IF NOT EXISTS memory_meta (
		memory_id TEXT PRIMARY KEY,
		memory_type TEXT NOT NULL DEFAULT '',
		access_count INTEGER NOT NULL DEFAULT 0,
		positive_count INTEGER NOT NULL DEFAULT 0,
		negative_count INTEGER NOT NULL DEFAULT 0,
		effectiveness REAL NOT NULL DEFAULT 0.5,
		last_accessed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

// RecordAccess records that a memory was accessed/used.
func (mm *MetaMemory) RecordAccess(memoryID, memoryType string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	now := time.Now().UTC()
	_, err := mm.db.Exec(`
		INSERT INTO memory_meta (memory_id, memory_type, access_count, last_accessed_at, created_at)
		VALUES (?, ?, 1, ?, ?)
		ON CONFLICT(memory_id) DO UPDATE SET
			access_count = access_count + 1,
			memory_type = CASE WHEN excluded.memory_type != '' THEN excluded.memory_type ELSE memory_meta.memory_type END,
			last_accessed_at = excluded.last_accessed_at
	`, memoryID, memoryType, now, now)
	return err
}

// RecordOutcome records whether using a memory led to a good or bad outcome
// and recalculates the effectiveness score.
func (mm *MetaMemory) RecordOutcome(memoryID string, positive bool) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	tx, err := mm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure the row exists.
	now := time.Now().UTC()
	_, err = tx.Exec(`
		INSERT INTO memory_meta (memory_id, memory_type, access_count, last_accessed_at, created_at)
		VALUES (?, '', 0, ?, ?)
		ON CONFLICT(memory_id) DO NOTHING
	`, memoryID, now, now)
	if err != nil {
		return err
	}

	// Update counts.
	if positive {
		_, err = tx.Exec(`UPDATE memory_meta SET positive_count = positive_count + 1 WHERE memory_id = ?`, memoryID)
	} else {
		_, err = tx.Exec(`UPDATE memory_meta SET negative_count = negative_count + 1 WHERE memory_id = ?`, memoryID)
	}
	if err != nil {
		return err
	}

	// Recalculate effectiveness.
	var pos, neg, access int
	var lastAccessed sql.NullTime
	err = tx.QueryRow(`SELECT positive_count, negative_count, access_count, last_accessed_at FROM memory_meta WHERE memory_id = ?`, memoryID).
		Scan(&pos, &neg, &access, &lastAccessed)
	if err != nil {
		return err
	}

	score := computeEffectiveness(pos, neg, access, lastAccessed.Time)

	_, err = tx.Exec(`UPDATE memory_meta SET effectiveness = ? WHERE memory_id = ?`, score, memoryID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// computeEffectiveness calculates a score from 0 to 1 based on outcomes and recency.
// The formula blends the outcome ratio with a recency decay factor.
func computeEffectiveness(positive, negative, accessCount int, lastAccessed time.Time) float64 {
	total := positive + negative
	if total == 0 {
		// No outcomes recorded yet; return neutral score with slight penalty for no data.
		return 0.5
	}

	// Base score: ratio of positive outcomes
	outcomeRatio := float64(positive) / float64(total)

	// Recency factor: decay over 30 days of inactivity (half-life ~30 days)
	daysSinceAccess := time.Since(lastAccessed).Hours() / 24
	if daysSinceAccess < 0 {
		daysSinceAccess = 0
	}
	recencyFactor := math.Exp(-0.023 * daysSinceAccess) // ~0.5 after 30 days

	// Usage factor: slight boost for frequently accessed memories (log scale, capped)
	usageFactor := 1.0
	if accessCount > 1 {
		usageFactor = math.Min(1.2, 1.0+0.1*math.Log2(float64(accessCount)))
	}

	score := outcomeRatio * recencyFactor * usageFactor
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}
	return score
}

// GetEffectiveness returns the effectiveness score for a memory.
// Returns 0 if the memory is not tracked.
func (mm *MetaMemory) GetEffectiveness(memoryID string) (float64, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	var score float64
	err := mm.db.QueryRow(`SELECT effectiveness FROM memory_meta WHERE memory_id = ?`, memoryID).Scan(&score)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return score, err
}

// GetTopMemories returns the most effective memories, ranked by effectiveness score descending.
func (mm *MetaMemory) GetTopMemories(limit int) ([]MemoryMeta, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := mm.db.Query(`
		SELECT memory_id, memory_type, access_count, positive_count, negative_count, effectiveness, last_accessed_at, created_at
		FROM memory_meta
		ORDER BY effectiveness DESC, access_count DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MemoryMeta
	for rows.Next() {
		var m MemoryMeta
		var lastAccessed sql.NullTime
		var createdAt sql.NullTime
		err := rows.Scan(&m.MemoryID, &m.MemoryType, &m.AccessCount,
			&m.PositiveCount, &m.NegativeCount, &m.Effectiveness,
			&lastAccessed, &createdAt)
		if err != nil {
			return nil, err
		}
		if lastAccessed.Valid {
			m.LastAccessedAt = lastAccessed.Time
		}
		if createdAt.Valid {
			m.CreatedAt = createdAt.Time
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// Prune returns memories with an effectiveness score below the given threshold.
// These are candidates for demotion or removal.
func (mm *MetaMemory) Prune(minScore float64) ([]MemoryMeta, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	rows, err := mm.db.Query(`
		SELECT memory_id, memory_type, access_count, positive_count, negative_count, effectiveness, last_accessed_at, created_at
		FROM memory_meta
		WHERE effectiveness < ?
		ORDER BY effectiveness ASC
	`, minScore)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MemoryMeta
	for rows.Next() {
		var m MemoryMeta
		var lastAccessed sql.NullTime
		var createdAt sql.NullTime
		err := rows.Scan(&m.MemoryID, &m.MemoryType, &m.AccessCount,
			&m.PositiveCount, &m.NegativeCount, &m.Effectiveness,
			&lastAccessed, &createdAt)
		if err != nil {
			return nil, err
		}
		if lastAccessed.Valid {
			m.LastAccessedAt = lastAccessed.Time
		}
		if createdAt.Valid {
			m.CreatedAt = createdAt.Time
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// AutoPromoteDemote recalculates effectiveness for all tracked memories
// and returns lists of promoted and demoted memory IDs.
// Memories with effectiveness >= promoteThreshold are promoted;
// memories with effectiveness < demoteThreshold are demoted.
func (mm *MetaMemory) AutoPromoteDemote(promoteThreshold, demoteThreshold float64) (promoted, demoted []string, err error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	rows, err := mm.db.Query(`
		SELECT memory_id, positive_count, negative_count, access_count, last_accessed_at, effectiveness
		FROM memory_meta
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	type entry struct {
		id       string
		newScore float64
		oldScore float64
	}
	var entries []entry

	for rows.Next() {
		var id string
		var pos, neg, access int
		var lastAccessed sql.NullTime
		var oldScore float64
		if err := rows.Scan(&id, &pos, &neg, &access, &lastAccessed, &oldScore); err != nil {
			return nil, nil, err
		}
		la := time.Time{}
		if lastAccessed.Valid {
			la = lastAccessed.Time
		}
		newScore := computeEffectiveness(pos, neg, access, la)
		entries = append(entries, entry{id: id, newScore: newScore, oldScore: oldScore})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	for _, e := range entries {
		// Update the score in the database.
		if _, err := mm.db.Exec(`UPDATE memory_meta SET effectiveness = ? WHERE memory_id = ?`, e.newScore, e.id); err != nil {
			return nil, nil, err
		}
		if e.newScore >= promoteThreshold {
			promoted = append(promoted, e.id)
		}
		if e.newScore < demoteThreshold {
			demoted = append(demoted, e.id)
		}
	}

	return promoted, demoted, nil
}
