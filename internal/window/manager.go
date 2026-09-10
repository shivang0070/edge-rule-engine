package window

import "sync"

type Manager struct {
	mu           sync.RWMutex
	stateWindows map[string]*StateWindow // ruleId -> window
	eventWindows map[string]*EventWindow // ruleId -> window
}

func NewManager() *Manager {
	return &Manager{
		stateWindows: make(map[string]*StateWindow),
		eventWindows: make(map[string]*EventWindow),
	}
}

func (m *Manager) GetOrCreateStateWindow(ruleId string, durationSeconds int) *StateWindow {
	m.mu.RLock()
	if w, ok := m.stateWindows[ruleId]; ok {
		m.mu.RUnlock()
		return w
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check locking
	if w, ok := m.stateWindows[ruleId]; ok {
		return w
	}
	
	w := NewStateWindow(durationSeconds)
	m.stateWindows[ruleId] = w
	return w
}

func (m *Manager) GetOrCreateEventWindow(ruleId string, durationSeconds int) *EventWindow {
	m.mu.RLock()
	if w, ok := m.eventWindows[ruleId]; ok {
		m.mu.RUnlock()
		return w
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check locking
	if w, ok := m.eventWindows[ruleId]; ok {
		return w
	}
	
	w := NewEventWindow(durationSeconds)
	m.eventWindows[ruleId] = w
	return w
}

func (m *Manager) RemoveWindows(ruleId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.stateWindows, ruleId)
	delete(m.eventWindows, ruleId)
}

func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stateWindows = make(map[string]*StateWindow)
	m.eventWindows = make(map[string]*EventWindow)
}
