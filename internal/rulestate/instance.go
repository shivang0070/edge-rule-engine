package rulestate

import (
	"context"
	"sync"
	"time"
)

type InstanceKey struct {
	RuleId         string
	CorrelationKey string
}

type StepMatch struct {
	StepIndex int
	MatchedAt int64
	Data      map[string]any
}

type Instance struct {
	Key          InstanceKey
	CreatedAt    int64
	Deadline     int64
	MatchedSteps []StepMatch
	Facts        map[string]any
}

type Store struct {
	mu        sync.RWMutex
	instances map[InstanceKey]*Instance
}

func NewStore(ctx context.Context) *Store {
	s := &Store{
		instances: make(map[InstanceKey]*Instance),
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

	now := time.Now().Unix()
	for k, inst := range s.instances {
		if inst.Deadline > 0 && now > inst.Deadline {
			delete(s.instances, k)
		}
	}
}

func (s *Store) GetOrCreate(key InstanceKey, deadline int64, now int64) *Instance {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, exists := s.instances[key]
	if !exists {
		inst = &Instance{
			Key:       key,
			CreatedAt: now,
			Deadline:  deadline,
			Facts:     make(map[string]any),
		}
		s.instances[key] = inst
	}
	return inst
}

func (s *Store) Complete(key InstanceKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.instances, key)
}
