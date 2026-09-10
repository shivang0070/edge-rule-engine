package fact

import (
	"sync"
	"edge-rule-engine/internal/observation"
)

type FactKey struct {
	CameraId    string
	ROIId       string
	MetricName  string
}

type Fact struct {
	Key         FactKey
	Value       any
	ComputedAt  int64
	BasedOn     int64
	Freshness   observation.Freshness
}

type FactStore struct {
	mu    sync.RWMutex
	facts map[FactKey]*Fact
}

func NewFactStore() *FactStore {
	return &FactStore{
		facts: make(map[FactKey]*Fact),
	}
}

func (s *FactStore) Set(fact *Fact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts[fact.Key] = fact
}

func (s *FactStore) Get(key FactKey) (*Fact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fact, ok := s.facts[key]
	return fact, ok
}
