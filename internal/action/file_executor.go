package action

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"edge-rule-engine/internal/model"
	"go.uber.org/zap"
)

type FileExecutor struct {
	directory string
	logger    *zap.Logger
}

func NewFileExecutor(directory string, logger *zap.Logger) (*FileExecutor, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create action directory: %w", err)
	}
	return &FileExecutor{
		directory: directory,
		logger:    logger,
	}, nil
}

func (e *FileExecutor) Execute(ctx context.Context, record model.ActionRecord) error {
	filename := fmt.Sprintf("action-%d-%s.json", record.TriggeredAt, record.RuleID)
	fullPath := filepath.Join(e.directory, filename)
	tempPath := fullPath + ".tmp"

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal action record: %w", err)
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp action file: %w", err)
	}

	if err := os.Rename(tempPath, fullPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp action file: %w", err)
	}

	e.logger.Info("executed file action", zap.String("rule_id", record.RuleID), zap.String("file", filename))
	return nil
}
