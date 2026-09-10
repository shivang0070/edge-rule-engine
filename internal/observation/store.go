package observation

import (
	"context"
	"sync"
	"time"
	"edge-rule-engine/internal/model"
)

type Freshness int

const (
	Fresh Freshness = iota
	Stale
	Missing
)

type Observation struct {
	CameraId   string
	ROIId      string
	Source     model.DataSourceKind
	EmittedAt  int64
	ReceivedAt int64
	
	Occupancy  model.Occupancy
	StaffCount int
	Dwell      map[string]model.DwellMetrics
	Entities   []model.Entity
	
	TTL        time.Duration
}

func (o *Observation) IsStale() bool {
	return time.Since(time.Unix(o.ReceivedAt, 0)) > o.TTL
}

func (o *Observation) ToEnv() map[string]any {
	dwellMap := make(map[string]any)
	for k, v := range o.Dwell {
		dwellMap[k] = map[string]any{
			"avg": v.Avg,
			"min": v.Min,
			"max": v.Max,
		}
	}

	return map[string]any{
		"occupancy": map[string]any{
			"person":  o.Occupancy.Person,
			"vehicle": o.Occupancy.Vehicle,
			"total":   o.Occupancy.Total,
		},
		"staffCount": o.StaffCount,
		"dwell":      dwellMap,
		"entities":   o.Entities,
	}
}

type Store struct {
	mu           sync.RWMutex
	observations map[string]*Observation
	defaultTTL   time.Duration
}

func NewStore(ctx context.Context, defaultTTL time.Duration) *Store {
	s := &Store{
		observations: make(map[string]*Observation),
		defaultTTL:   defaultTTL,
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.Cleanup()
			}
		}
	}()

	return s
}

func (s *Store) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, obs := range s.observations {
		if obs.IsStale() {
			delete(s.observations, k)
		}
	}
}

func (s *Store) key(cameraId, roiId string) string {
	return cameraId + ":" + roiId
}

func (s *Store) Update(cameraId, roiId string, region model.Region, emittedAt int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := s.key(cameraId, roiId)
	s.observations[k] = &Observation{
		CameraId:   cameraId,
		ROIId:      roiId,
		Source:     model.SourceState,
		EmittedAt:  emittedAt,
		ReceivedAt: time.Now().Unix(),
		Occupancy:  region.Occupancy,
		StaffCount: region.StaffCount,
		Dwell:      region.Precalc.Dwell,
		Entities:   region.Entities,
		TTL:        s.defaultTTL,
	}
}

func (s *Store) Get(cameraId, roiId string) (*Observation, Freshness) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obs, exists := s.observations[s.key(cameraId, roiId)]
	if !exists {
		return nil, Missing
	}

	if obs.IsStale() {
		return obs, Stale
	}

	return obs, Fresh
}
