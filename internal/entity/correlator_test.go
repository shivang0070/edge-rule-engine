package entity

import (
	"context"
	"testing"
	"time"
	"edge-rule-engine/internal/model"
	"go.uber.org/zap"
)

func TestEntityCorrelation_AcrossCameras(t *testing.T) {
	store := NewStore(context.Background(), 1 * time.Hour)
	correlator := NewCorrelator(store, zap.NewNop())
	
	now := time.Now().Unix()

	// Entity 1 only in Cam1
	store.UpdateFromState("Cam1", "ZoneA", []model.Entity{
		{TrackId: "PersonA", EntityType: "person"},
	}, now)

	// Entity 2 in Cam1 AND Cam2
	store.UpdateFromState("Cam1", "ZoneA", []model.Entity{
		{TrackId: "PersonB", EntityType: "person"},
	}, now)
	store.UpdateFromState("Cam2", "ZoneB", []model.Entity{
		{TrackId: "PersonB", EntityType: "person"},
	}, now+5)

	rule := model.Rule{
		Correlation: &model.Correlation{
			Key:    "trackId",
			Across: "cameras",
		},
	}

	keys := []EntityKey{
		{TrackId: "PersonA", EntityType: "person"},
		{TrackId: "PersonB", EntityType: "person"},
	}

	matches := correlator.FindCorrelated(rule, keys)
	
	if len(matches) != 2 {
		t.Fatalf("Expected 2 matches returned, got %d", len(matches))
	}
	
	if matches[0].IsMatch {
		t.Errorf("PersonA should not match cross-camera correlation")
	}
	
	if !matches[1].IsMatch {
		t.Errorf("PersonB SHOULD match cross-camera correlation")
	}
}
