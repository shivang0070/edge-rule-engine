package replay

import (
	"context"
	"edge-rule-engine/internal/store"
	"edge-rule-engine/internal/engine"
	"go.uber.org/zap"
)

type Replayer struct {
	db     *store.StateStore
	engine *engine.Engine
	logger *zap.Logger
}

func NewReplayer(db *store.StateStore, eng *engine.Engine, logger *zap.Logger) *Replayer {
	return &Replayer{
		db:     db,
		engine: eng,
		logger: logger,
	}
}

func (r *Replayer) ReplayRule(ctx context.Context, ruleId string, start int64, end int64) error {
	// 1. Unregister temporarily
	// 2. Fetch all historical states from db between start and end
	// 3. Feed them to a sandboxed engine instance
	// 4. Return triggered actions without executing them
	return nil
}
