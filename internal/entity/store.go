package entity

import (
	"context"
	"sync"
	"time"
	"edge-rule-engine/internal/model"
)

type EntityKey struct {
	TrackId    string
	EntityType string
}

type EntityEvent struct {
	Timestamp int64
	CameraId  string
	ROIId     string
	EventType string // "detected", "entered", "exited"
}

type EntityState struct {
	Key           EntityKey
	FirstSeen     int64
	LastSeen      int64
	CurrentROI    string
	PreviousROI   string
	CurrentCamera string
	Cameras       map[string]int64
	ROIs          map[string]int64
	Attributes    map[string]string
	DwellTime     float64
	History       []EntityEvent
}

type Store struct {
	mu       sync.RWMutex
	entities map[EntityKey]*EntityState
	ttl      time.Duration
}

func NewStore(ctx context.Context, ttl time.Duration) *Store {
	s := &Store{
		entities: make(map[EntityKey]*EntityState),
		ttl:      ttl,
	}

	// Background garbage collection to prevent memory leaks from unbounded TrackIds
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
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
	ttlSeconds := int64(s.ttl.Seconds())

	for k, state := range s.entities {
		if now-state.LastSeen > ttlSeconds {
			delete(s.entities, k)
		}
	}
}

func (s *Store) UpdateFromState(cameraId, roiId string, entities []model.Entity, emittedAt int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ent := range entities {
		key := EntityKey{TrackId: ent.TrackId, EntityType: ent.EntityType}
		state, exists := s.entities[key]
		
		if !exists {
			state = &EntityState{
				Key:           key,
				FirstSeen:     emittedAt,
				Cameras:       make(map[string]int64),
				ROIs:          make(map[string]int64),
				Attributes:    make(map[string]string),
			}
			s.entities[key] = state
		}

		if state.CurrentROI != roiId && state.CurrentROI != "" {
			state.PreviousROI = state.CurrentROI
		}
		
		state.LastSeen = emittedAt
		state.CurrentCamera = cameraId
		state.CurrentROI = roiId
		state.Cameras[cameraId] = emittedAt
		state.ROIs[roiId] = emittedAt
		
		if ent.DwellTime != nil {
			state.DwellTime = *ent.DwellTime
		}
		
		for k, v := range ent.Attributes {
			state.Attributes[k] = v
		}
	}
}

func (s *Store) UpdateFromEvent(event model.EventPayload) {
	if event.Entity == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := EntityKey{TrackId: event.Entity.TrackId, EntityType: event.Entity.EntityType}
	state, exists := s.entities[key]
	
	if !exists {
		state = &EntityState{
			Key:           key,
			FirstSeen:     event.EmittedAt,
			Cameras:       make(map[string]int64),
			ROIs:          make(map[string]int64),
			Attributes:    make(map[string]string),
		}
		s.entities[key] = state
	}

	state.LastSeen = event.EmittedAt
	
	historyEvent := EntityEvent{
		Timestamp: event.EmittedAt,
		CameraId:  event.Camera,
		ROIId:     event.ROIId,
		EventType: event.Kind,
	}
	
	state.History = append(state.History, historyEvent)
	
	if len(state.History) > 50 {
		state.History = state.History[1:]
	}
}

func (s *Store) Get(key EntityKey) *EntityState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entities[key]
}
