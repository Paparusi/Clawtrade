package database

import "database/sql"

func migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS trade_episodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			symbol TEXT NOT NULL,
			side TEXT NOT NULL,
			entry_price REAL,
			exit_price REAL,
			size REAL,
			pnl REAL,
			exchange TEXT,
			strategy TEXT,
			reasoning TEXT,
			outcome TEXT,
			emotion_tag TEXT,
			confidence REAL,
			post_mortem TEXT,
			opened_at DATETIME,
			closed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS semantic_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			category TEXT,
			confidence REAL DEFAULT 0.5,
			evidence_count INTEGER DEFAULT 0,
			effectiveness REAL DEFAULT 0.5,
			source TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_profile (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			details TEXT,
			reasoning TEXT,
			risk_check TEXT,
			permission TEXT,
			prev_hash TEXT,
			hash TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS knowledge_graph (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_from TEXT NOT NULL,
			relation TEXT NOT NULL,
			entity_to TEXT NOT NULL,
			weight REAL DEFAULT 1.0,
			evidence TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_episodes_symbol ON trade_episodes(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_episodes_opened ON trade_episodes(opened_at)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_category ON semantic_rules(category)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action)`,
		`CREATE INDEX IF NOT EXISTS idx_graph_from ON knowledge_graph(entity_from)`,
		`CREATE INDEX IF NOT EXISTS idx_graph_to ON knowledge_graph(entity_to)`,
		`CREATE TABLE IF NOT EXISTS candle_cache (
			symbol TEXT NOT NULL,
			timeframe TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			open REAL NOT NULL,
			high REAL NOT NULL,
			low REAL NOT NULL,
			close REAL NOT NULL,
			volume REAL NOT NULL,
			PRIMARY KEY (symbol, timeframe, timestamp)
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			symbol TEXT,
			condition TEXT NOT NULL,
			threshold REAL,
			expression TEXT,
			message TEXT,
			enabled BOOLEAN DEFAULT 1,
			one_shot BOOLEAN DEFAULT 0,
			last_triggered_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			alert_id INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			value REAL,
			message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}

	return nil
}
