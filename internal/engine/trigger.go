package engine

import (
	"sync"
	"edge-rule-engine/internal/model"
	"go.uber.org/zap"
)

type TriggerState struct {
	mu               sync.Mutex
	PreviousResult   bool
	SustainStartTime int64
	LastTriggeredAt  int64
}

type TriggerManager struct {
	mu     sync.RWMutex
	states map[string]*TriggerState
	logger *zap.Logger
}

func NewTriggerManager(logger *zap.Logger) *TriggerManager {
	return &TriggerManager{
		states: make(map[string]*TriggerState),
		logger: logger,
	}
}

func (tm *TriggerManager) getOrCreateState(ruleId string) *TriggerState {
	tm.mu.RLock()
	state, ok := tm.states[ruleId]
	tm.mu.RUnlock()
	if ok {
		return state
	}
	
	tm.mu.Lock()
	defer tm.mu.Unlock()
	state, ok = tm.states[ruleId]
	if !ok {
		state = &TriggerState{}
		tm.states[ruleId] = state
	}
	return state
}

func (tm *TriggerManager) ShouldTrigger(ruleId string, conditionResult bool, emittedAt int64, trigger model.TriggerConfig, sustain *model.SustainConfig) bool {
	state := tm.getOrCreateState(ruleId)
	
	state.mu.Lock()
	defer state.mu.Unlock()

	if !conditionResult {
		state.SustainStartTime = 0
		state.PreviousResult = false
		return false
	}

	if sustain != nil && sustain.DurationSeconds > 0 {
		if state.SustainStartTime == 0 {
			state.SustainStartTime = emittedAt
			state.PreviousResult = true
			return false
		}
		if emittedAt-state.SustainStartTime < int64(sustain.DurationSeconds) {
			state.PreviousResult = true
			return false
		}
	}

	mode := trigger.Mode
	if mode == "" {
		mode = model.TriggerModeRising
	}

	shouldTrigger := false
	if mode == model.TriggerModeRising {
		if sustain != nil && sustain.DurationSeconds > 0 {
			shouldTrigger = true
		} else {
			shouldTrigger = !state.PreviousResult
		}
	} else if mode == model.TriggerModeEvery {
		shouldTrigger = true
	}

	if !shouldTrigger {
		state.PreviousResult = true
		return false
	}

	if state.LastTriggeredAt > 0 && emittedAt-state.LastTriggeredAt < int64(trigger.CooldownSeconds) {
		state.PreviousResult = true
		return false
	}

	state.LastTriggeredAt = emittedAt
	state.PreviousResult = true
	state.SustainStartTime = 0
	return true
}

func (tm *TriggerManager) Reset(ruleId string) {
	tm.Remove(ruleId)
}

func (tm *TriggerManager) Remove(ruleId string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.states, ruleId)
}

func (tm *TriggerManager) GetState(ruleId string) *TriggerState {
	tm.mu.RLock()
	state, ok := tm.states[ruleId]
	tm.mu.RUnlock()
	if ok {
		state.mu.Lock()
		defer state.mu.Unlock()
		s := *state
		return &s
	}
	return nil
}
