package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// NewDB opens a new SQLite database connection and runs migrations.
func NewDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// RunMigrations creates the necessary database tables and indices.
func RunMigrations(db *sql.DB) error {
	createRulesTable := `
	CREATE TABLE IF NOT EXISTS task_rules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		enabled INTEGER NOT NULL DEFAULT 1,
		camera_id TEXT NOT NULL,
		roi_id TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL,
		rule_json TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(createRulesTable); err != nil {
		return err
	}

	createVersionsTable := `
	CREATE TABLE IF NOT EXISTS rule_versions (
		rule_id TEXT NOT NULL,
		version INTEGER NOT NULL,
		rule_json TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		PRIMARY KEY (rule_id, version)
	);
	`
	if _, err := db.Exec(createVersionsTable); err != nil {
		return err
	}

	migrations := `
CREATE TABLE IF NOT EXISTS states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    camera_id TEXT NOT NULL,
    emitted_at INTEGER NOT NULL,
    payload TEXT NOT NULL,
    received_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_states_camera_emitted ON states(camera_id, emitted_at);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE NOT NULL,
    camera_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    emitted_at INTEGER NOT NULL,
    payload TEXT NOT NULL,
    received_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_events_camera_emitted ON events(camera_id, emitted_at);
CREATE INDEX IF NOT EXISTS idx_events_event_id ON events(event_id);

CREATE TABLE IF NOT EXISTS rule_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id TEXT NOT NULL,
    triggered_at INTEGER NOT NULL,
    status TEXT NOT NULL,
    action_type TEXT NOT NULL,
    context TEXT,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_rule_executions_rule_id ON rule_executions(rule_id, triggered_at);
`
	_, err := db.Exec(migrations)
	return err
}
