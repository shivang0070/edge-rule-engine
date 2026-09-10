package entity

import (
	"edge-rule-engine/internal/model"
	"go.uber.org/zap"
)

type Correlator struct {
	store  *Store
	logger *zap.Logger
}

func NewCorrelator(store *Store, logger *zap.Logger) *Correlator {
	return &Correlator{
		store:  store,
		logger: logger,
	}
}

type CorrelatedMatch struct {
	EntityKey EntityKey
	IsMatch   bool
}

func (c *Correlator) FindCorrelated(rule model.Rule, currentEntities []EntityKey) []CorrelatedMatch {
	if rule.Correlation == nil {
		// If no correlation rule, everything matches
		matches := make([]CorrelatedMatch, len(currentEntities))
		for i, e := range currentEntities {
			matches[i] = CorrelatedMatch{EntityKey: e, IsMatch: true}
		}
		return matches
	}

	var matches []CorrelatedMatch
	
	for _, key := range currentEntities {
		state := c.store.Get(key)
		if state == nil {
			matches = append(matches, CorrelatedMatch{EntityKey: key, IsMatch: false})
			continue
		}
		
		isMatch := false
		if rule.Correlation.Across == "cameras" {
			// Must be seen in more than 1 camera
			if len(state.Cameras) > 1 {
				isMatch = true
			}
		} else if rule.Correlation.Across == "rois" {
			// Must be seen in more than 1 ROI
			if len(state.ROIs) > 1 {
				isMatch = true
			}
		}

		matches = append(matches, CorrelatedMatch{EntityKey: key, IsMatch: isMatch})
	}
	
	return matches
}
