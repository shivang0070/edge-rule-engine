package engine

import (
	"sync"
	"edge-rule-engine/internal/model"
)

type RuleIndex struct {
	mu         sync.RWMutex
	stateRules map[string]map[string][]*ActiveRule
	eventRules map[string]map[string][]*ActiveRule
}

func NewRuleIndex() *RuleIndex {
	return &RuleIndex{
		stateRules: make(map[string]map[string][]*ActiveRule),
		eventRules: make(map[string]map[string][]*ActiveRule),
	}
}

func (i *RuleIndex) Add(rule *ActiveRule) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if rule.Rule.Source == model.SourceState {
		cam := rule.Rule.Scope.CameraId
		roi := rule.Rule.Scope.ROIId
		if i.stateRules[cam] == nil {
			i.stateRules[cam] = make(map[string][]*ActiveRule)
		}
		i.stateRules[cam][roi] = append(i.stateRules[cam][roi], rule)
	} else if rule.Rule.Source == model.SourceEvent {
		cam := rule.Rule.Scope.CameraId
		kind := rule.Rule.Scope.EventKind
		if i.eventRules[cam] == nil {
			i.eventRules[cam] = make(map[string][]*ActiveRule)
		}
		i.eventRules[cam][kind] = append(i.eventRules[cam][kind], rule)
	}
}

func (i *RuleIndex) Remove(ruleId string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	for cam, rois := range i.stateRules {
		for roi, rules := range rois {
			var newRules []*ActiveRule
			for _, r := range rules {
				if r.Rule.ID != ruleId {
					newRules = append(newRules, r)
				}
			}
			if len(newRules) == 0 {
				delete(i.stateRules[cam], roi)
			} else {
				i.stateRules[cam][roi] = newRules
			}
		}
		if len(i.stateRules[cam]) == 0 {
			delete(i.stateRules, cam)
		}
	}

	for cam, kinds := range i.eventRules {
		for kind, rules := range kinds {
			var newRules []*ActiveRule
			for _, r := range rules {
				if r.Rule.ID != ruleId {
					newRules = append(newRules, r)
				}
			}
			if len(newRules) == 0 {
				delete(i.eventRules[cam], kind)
			} else {
				i.eventRules[cam][kind] = newRules
			}
		}
		if len(i.eventRules[cam]) == 0 {
			delete(i.eventRules, cam)
		}
	}
}

func (i *RuleIndex) FindStateRules(cameraId, roiId string) []*ActiveRule {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var result []*ActiveRule
	if camMap, ok := i.stateRules[cameraId]; ok {
		if rules, ok := camMap[roiId]; ok {
			result = append(result, rules...)
		}
	}
	return result
}

func (i *RuleIndex) FindEventRules(cameraId, eventKind string) []*ActiveRule {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var result []*ActiveRule
	if camMap, ok := i.eventRules[cameraId]; ok {
		if rules, ok := camMap[eventKind]; ok {
			result = append(result, rules...)
		}
		if rules, ok := camMap[""]; ok {
			result = append(result, rules...)
		}
		if rules, ok := camMap["*"]; ok && eventKind != "*" {
			result = append(result, rules...)
		}
	}
	return result
}
