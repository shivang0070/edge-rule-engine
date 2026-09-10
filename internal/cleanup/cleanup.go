package cleanup

import (
	"context"
	"time"

	"edge-rule-engine/internal/store"

	"go.uber.org/zap"
)

type Cleaner struct {
	stateStore          store.StateStore
	eventStore          store.EventStore
	stateRetentionHours int
	eventRetentionHours int
	logger              *zap.Logger
}

func NewCleaner(
	stateStore store.StateStore,
	eventStore store.EventStore,
	stateRetentionHours,
	eventRetentionHours int,
	logger *zap.Logger,
) *Cleaner {
	return &Cleaner{
		stateStore:          stateStore,
		eventStore:          eventStore,
		stateRetentionHours: stateRetentionHours,
		eventRetentionHours: eventRetentionHours,
		logger:              logger,
	}
}

func (c *Cleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	c.RunOnce()

	for {
		select {
		case <-ticker.C:
			c.RunOnce()
		case <-ctx.Done():
			c.logger.Info("cleanup worker stopped")
			return
		}
	}
}

func (c *Cleaner) RunOnce() {
	now := time.Now()
	stateThreshold := now.Add(-time.Duration(c.stateRetentionHours) * time.Hour).Unix()
	eventThreshold := now.Add(-time.Duration(c.eventRetentionHours) * time.Hour).Unix()



	deletedStates, err := c.stateStore.Cleanup(stateThreshold)
	if err != nil {
		c.logger.Error("failed to clean up old states", zap.Error(err))
	} else {
		c.logger.Info("cleaned up old states", zap.Int64("deleted", deletedStates))
	}

	deletedEvents, err := c.eventStore.Cleanup(eventThreshold)
	if err != nil {
		c.logger.Error("failed to clean up old events", zap.Error(err))
	} else {
		c.logger.Info("cleaned up old events", zap.Int64("deleted", deletedEvents))
	}
}
