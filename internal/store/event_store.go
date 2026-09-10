package store

import (
	"database/sql"
	"fmt"
	"time"
)

type EventStore interface {
	Save(eventId, cameraId, kind string, emittedAt int64, payload []byte) error
	Exists(eventId string) (bool, error)
	Cleanup(olderThan int64) (int64, error)
}

type SQLiteEventStore struct {
	db *sql.DB
}

func NewEventStore(db *sql.DB) EventStore {
	return &SQLiteEventStore{db: db}
}

func (s *SQLiteEventStore) Save(eventId, cameraId, kind string, emittedAt int64, payload []byte) error {
	query := `
		INSERT OR IGNORE INTO events (event_id, camera_id, kind, emitted_at, payload, received_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	receivedAt := time.Now().Unix()
	_, err := s.db.Exec(query, eventId, cameraId, kind, emittedAt, string(payload), receivedAt)
	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}
	return nil
}

func (s *SQLiteEventStore) Exists(eventId string) (bool, error) {
	query := `SELECT 1 FROM events WHERE event_id = ?`
	var exists int
	err := s.db.QueryRow(query, eventId).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check event existence: %w", err)
	}
	return true, nil
}

func (s *SQLiteEventStore) Cleanup(olderThan int64) (int64, error) {
	query := `DELETE FROM events WHERE emitted_at < ?`
	res, err := s.db.Exec(query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup events: %w", err)
	}
	
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted rows count: %w", err)
	}
	return rowsAffected, nil
}
