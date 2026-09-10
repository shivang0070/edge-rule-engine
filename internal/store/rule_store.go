package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"edge-rule-engine/internal/model"
)

type RuleStore interface {
	Create(rule *model.Rule) error
	GetByID(id string) (*model.Rule, error)
	List() ([]*model.Rule, error)
	Update(rule *model.Rule) error
	Delete(id string) error
}

type SQLiteRuleStore struct {
	db *sql.DB
}

func NewRuleStore(db *sql.DB) RuleStore {
	return &SQLiteRuleStore{db: db}
}

func (s *SQLiteRuleStore) Create(rule *model.Rule) error {
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	enabledInt := 0
	if rule.Enabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO task_rules (id, name, enabled, camera_id, roi_id, source, rule_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(query,
		rule.ID,
		rule.Name,
		enabledInt,
		rule.Scope.CameraId,
		rule.Scope.ROIId,
		rule.Source,
		string(ruleJSON),
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert rule: %w", err)
	}
	return nil
}

func (s *SQLiteRuleStore) GetByID(id string) (*model.Rule, error) {
	query := `SELECT rule_json FROM task_rules WHERE id = ?`
	
	var ruleJSON string
	err := s.db.QueryRow(query, id).Scan(&ruleJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("rule not found")
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	var rule model.Rule
	if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	return &rule, nil
}

func (s *SQLiteRuleStore) List() ([]*model.Rule, error) {
	query := `SELECT rule_json FROM task_rules`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []*model.Rule
	for rows.Next() {
		var ruleJSON string
		if err := rows.Scan(&ruleJSON); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		var rule model.Rule
		if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
		}
		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return rules, nil
}

func (s *SQLiteRuleStore) Update(rule *model.Rule) error {
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	enabledInt := 0
	if rule.Enabled {
		enabledInt = 1
	}

	query := `
		UPDATE task_rules 
		SET name = ?, enabled = ?, camera_id = ?, roi_id = ?, source = ?, rule_json = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := s.db.Exec(query,
		rule.Name,
		enabledInt,
		rule.Scope.CameraId,
		rule.Scope.ROIId,
		rule.Source,
		string(ruleJSON),
		rule.UpdatedAt,
		rule.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

func (s *SQLiteRuleStore) Delete(id string) error {
	query := `DELETE FROM task_rules WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	return nil
}
