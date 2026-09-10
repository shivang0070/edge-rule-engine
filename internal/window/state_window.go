package window

import "sync"

// StateEntry represents a single state snapshot in the window
type StateEntry struct {
	EmittedAt  int64
	CameraId   string
	ROIId      string
	Occupancy  OccupancyData
	StaffCount int
	Dwell      map[string]DwellData // keyed by entity type: "person", "vehicle"
	Entities   []EntityData
}

type OccupancyData struct {
	Total   int
	Person  int
	Vehicle int
}

type DwellData struct {
	Avg float64
	Min float64
	Max float64
}

type EntityData struct {
	EntityType string
	TrackId    string
	DwellTime  float64
}

type StateWindow struct {
	mu              sync.RWMutex
	entries         []StateEntry
	durationSeconds int64
	latest          int64
}

func NewStateWindow(durationSeconds int) *StateWindow {
	return &StateWindow{
		entries:         make([]StateEntry, 0),
		durationSeconds: int64(durationSeconds),
	}
}

func (w *StateWindow) Add(entry StateEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries = append(w.entries, entry)
	
	if entry.EmittedAt > w.latest {
		w.latest = entry.EmittedAt
	}

	cutoff := w.latest - w.durationSeconds
	w.pruneLocked(cutoff)
}

func (w *StateWindow) Prune(cutoff int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pruneLocked(cutoff)
}

func (w *StateWindow) pruneLocked(cutoff int64) {
	n := 0
	for _, e := range w.entries {
		if e.EmittedAt >= cutoff {
			w.entries[n] = e
			n++
		}
	}
	
	// Clear stale pointers to help GC
	for i := n; i < len(w.entries); i++ {
		w.entries[i] = StateEntry{}
	}
	
	w.entries = w.entries[:n]
}

func (w *StateWindow) Len() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.entries)
}

func (w *StateWindow) Entries() []StateEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	copyEntries := make([]StateEntry, len(w.entries))
	copy(copyEntries, w.entries)
	return copyEntries
}

func (w *StateWindow) UniqueTrackIds(entityType string) map[string]struct{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	unique := make(map[string]struct{})
	for _, entry := range w.entries {
		for _, entity := range entry.Entities {
			if entity.EntityType == entityType {
				unique[entity.TrackId] = struct{}{}
			}
		}
	}
	return unique
}

func (w *StateWindow) LatestEntry() (StateEntry, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.entries) == 0 {
		return StateEntry{}, false
	}
	
	latest := w.entries[0]
	for _, e := range w.entries[1:] {
		if e.EmittedAt > latest.EmittedAt {
			latest = e
		}
	}
	
	return latest, true
}

func (w *StateWindow) Aggregate(agg string, field string) any {
	w.mu.RLock()
	defer w.mu.RUnlock()

	switch agg {
	case "count":
		return len(w.entries)
	case "uniqueCount":
		if field == "trackId.person" {
			return len(w.UniqueTrackIds("person")) // V1 compat
		}
		if field == "trackId.vehicle" {
			return len(w.UniqueTrackIds("vehicle")) // V1 compat
		}
	}
	return 0
}
