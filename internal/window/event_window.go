package window

import "sync"

type EventEntry struct {
	EventId    string
	Kind       string
	EmittedAt  int64
	CameraId   string
	ROIId      string
	TrackId    string
	EntityType string
}

type EventWindow struct {
	mu              sync.RWMutex
	entries         []EventEntry
	seenEventIds    map[string]struct{}
	durationSeconds int64
	latest          int64
}

func NewEventWindow(durationSeconds int) *EventWindow {
	return &EventWindow{
		entries:         make([]EventEntry, 0),
		seenEventIds:    make(map[string]struct{}),
		durationSeconds: int64(durationSeconds),
	}
}

func (w *EventWindow) Add(entry EventEntry) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.seenEventIds[entry.EventId]; exists {
		return false
	}

	w.entries = append(w.entries, entry)
	w.seenEventIds[entry.EventId] = struct{}{}

	if entry.EmittedAt > w.latest {
		w.latest = entry.EmittedAt
	}

	cutoff := w.latest - w.durationSeconds
	w.pruneLocked(cutoff)
	return true
}

func (w *EventWindow) Prune(cutoff int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pruneLocked(cutoff)
}

func (w *EventWindow) pruneLocked(cutoff int64) {
	n := 0
	for _, e := range w.entries {
		if e.EmittedAt >= cutoff {
			w.entries[n] = e
			n++
		} else {
			delete(w.seenEventIds, e.EventId)
		}
	}
	
	for i := n; i < len(w.entries); i++ {
		w.entries[i] = EventEntry{}
	}
	
	w.entries = w.entries[:n]
}

func (w *EventWindow) Count() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.entries)
}

func (w *EventWindow) CountByKind(kind string) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	count := 0
	for _, e := range w.entries {
		if e.Kind == kind {
			count++
		}
	}
	return count
}

func (w *EventWindow) CountByFilter(fn func(EventEntry) bool) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	count := 0
	for _, e := range w.entries {
		if fn(e) {
			count++
		}
	}
	return count
}

func (w *EventWindow) Entries() []EventEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	copyEntries := make([]EventEntry, len(w.entries))
	copy(copyEntries, w.entries)
	return copyEntries
}

func (w *EventWindow) Aggregate(agg string) any {
	w.mu.RLock()
	defer w.mu.RUnlock()

	switch agg {
	case "count":
		return len(w.entries)
	}
	return 0
}
