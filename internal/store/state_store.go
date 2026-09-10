package store

import (
	"database/sql"
	"fmt"
	"time"
)

type StateStore interface {
	Save(cameraId string, emittedAt int64, payload []byte) error
	Cleanup(olderThan int64) (int64, error)
}

type SQLiteStateStore struct {
	db *sql.DB
}

func NewStateStore(db *sql.DB) StateStore {
	return &SQLiteStateStore{db: db}
}

func (s *SQLiteStateStore) Save(cameraId string, emittedAt int64, payload []byte) error {
	query := `
		INSERT INTO states (camera_id, emitted_at, payload, received_at)
		VALUES (?, ?, ?, ?)
	`
	receivedAt := time.Now().Unix()
	_, err := s.db.Exec(query, cameraId, emittedAt, string(payload), receivedAt)
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	return nil
}

func (s *SQLiteStateStore) Cleanup(olderThan int64) (int64, error) {
	query := `DELETE FROM states WHERE emitted_at < ?`
	res, err := s.db.Exec(query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup states: %w", err)
	}
	
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted rows count: %w", err)
	}
	return rowsAffected, nil
}
