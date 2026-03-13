package security

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AuditLog provides a tamper-evident, hash-chained audit trail for all
// security-relevant actions in the system.
type AuditLog struct {
	db       *sql.DB
	lastHash string
}

// AuditEntry represents a single row in the audit_log table.
type AuditEntry struct {
	ID        int64          `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Details   map[string]any `json:"details,omitempty"`
	Reasoning string         `json:"reasoning,omitempty"`
	PrevHash  string         `json:"prev_hash"`
	Hash      string         `json:"hash"`
	CreatedAt time.Time      `json:"created_at"`
}

// NewAuditLog creates an AuditLog that writes to db.  It reads the last
// hash from the database so new entries are chained correctly.
func NewAuditLog(db *sql.DB) *AuditLog {
	al := &AuditLog{db: db}
	var hash sql.NullString
	db.QueryRow("SELECT hash FROM audit_log ORDER BY id DESC LIMIT 1").Scan(&hash)
	if hash.Valid {
		al.lastHash = hash.String
	}
	return al
}

// Log appends a new entry to the audit chain.
func (al *AuditLog) Log(actor, action string, details map[string]any, reasoning string) error {
	detailsJSON, _ := json.Marshal(details)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		al.lastHash, actor, action, string(detailsJSON), reasoning, now)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))

	_, err := al.db.Exec(
		`INSERT INTO audit_log (actor, action, details, reasoning, prev_hash, hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		actor, action, string(detailsJSON), reasoning, al.lastHash, hash, now,
	)
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}

	al.lastHash = hash
	return nil
}

// VerifyChain walks every entry in order and recomputes hashes.  It returns
// false (without error) when tampering is detected.
func (al *AuditLog) VerifyChain() (bool, error) {
	rows, err := al.db.Query(
		"SELECT id, actor, action, details, reasoning, prev_hash, hash, created_at FROM audit_log ORDER BY id ASC",
	)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	prevHash := ""
	for rows.Next() {
		var entry AuditEntry
		var detailsStr string
		var createdAt string
		err := rows.Scan(&entry.ID, &entry.Actor, &entry.Action, &detailsStr,
			&entry.Reasoning, &entry.PrevHash, &entry.Hash, &createdAt)
		if err != nil {
			return false, err
		}

		if entry.PrevHash != prevHash {
			return false, nil
		}

		hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
			entry.PrevHash, entry.Actor, entry.Action, detailsStr, entry.Reasoning, createdAt)
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))

		if entry.Hash != expectedHash {
			return false, nil
		}

		prevHash = entry.Hash
	}

	return true, nil
}

// Query returns the most recent entries, optionally filtered by action.
func (al *AuditLog) Query(action string, limit int) ([]AuditEntry, error) {
	query := "SELECT id, actor, action, details, reasoning, prev_hash, hash, created_at FROM audit_log"
	var args []any
	if action != "" {
		query += " WHERE action = ?"
		args = append(args, action)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := al.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var detailsStr, createdAt string
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &detailsStr,
			&e.Reasoning, &e.PrevHash, &e.Hash, &createdAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(detailsStr), &e.Details)
		t, _ := time.Parse(time.RFC3339Nano, createdAt)
		e.CreatedAt = t
		entries = append(entries, e)
	}
	return entries, nil
}
