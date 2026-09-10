package store

import (
	"database/sql"
	"fmt"

	"edge-rule-engine/internal/model"
)

type ExecutionStore interface {
	Record(exec *model.RuleExecution) error
	ListByRule(ruleId string, limit int) ([]*model.RuleExecution, error)
	ListAll(limit int) ([]*model.RuleExecution, error)
}

type SQLiteExecutionStore struct {
	db *sql.DB
}

func NewExecutionStore(db *sql.DB) ExecutionStore {
	return &SQLiteExecutionStore{db: db}
}

func (s *SQLiteExecutionStore) Record(exec *model.RuleExecution) error {
	query := `
		INSERT INTO rule_executions (rule_id, triggered_at, status, action_type, context, error)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	// Convert optional fields to nullable SQL fields
	var context, errStr sql.NullString
	if exec.Context != "" {
		context.String = exec.Context
		context.Valid = true
	}
	if exec.Error != "" {
		errStr.String = exec.Error
		errStr.Valid = true
	}

	res, err := s.db.Exec(query, 
		exec.RuleID, 
		exec.TriggeredAt, 
		exec.Status, 
		exec.ActionType, 
		context, 
		errStr,
	)
	if err != nil {
		return fmt.Errorf("failed to record execution: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	
	exec.ID = id
	return nil
}

func (s *SQLiteExecutionStore) ListByRule(ruleId string, limit int) ([]*model.RuleExecution, error) {
	query := `
		SELECT id, rule_id, triggered_at, status, action_type, context, error 
		FROM rule_executions 
		WHERE rule_id = ? 
		ORDER BY triggered_at DESC 
		LIMIT ?
	`
	rows, err := s.db.Query(query, ruleId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.RuleExecution
	for rows.Next() {
		var exec model.RuleExecution
		var ctx sql.NullString
		var errStr sql.NullString

		err := rows.Scan(
			&exec.ID,
			&exec.RuleID,
			&exec.TriggeredAt,
			&exec.Status,
			&exec.ActionType,
			&ctx,
			&errStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		if ctx.Valid {
			exec.Context = ctx.String
		}
		if errStr.Valid {
			exec.Error = errStr.String
		}

		executions = append(executions, &exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return executions, nil
}

func (s *SQLiteExecutionStore) ListAll(limit int) ([]*model.RuleExecution, error) {
	query := `
		SELECT id, rule_id, triggered_at, status, action_type, context, error 
		FROM rule_executions 
		ORDER BY triggered_at DESC 
		LIMIT ?
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.RuleExecution
	for rows.Next() {
		var exec model.RuleExecution
		var ctx sql.NullString
		var errStr sql.NullString

		err := rows.Scan(
			&exec.ID,
			&exec.RuleID,
			&exec.TriggeredAt,
			&exec.Status,
			&exec.ActionType,
			&ctx,
			&errStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		if ctx.Valid {
			exec.Context = ctx.String
		}
		if errStr.Valid {
			exec.Error = errStr.String
		}

		executions = append(executions, &exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return executions, nil
}
