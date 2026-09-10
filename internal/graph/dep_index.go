package graph

import (
	"sync"
	"edge-rule-engine/internal/model"
)

// DepKey uniquely identifies a data stream
type DepKey struct {
	CameraId  string
	ROIId     string
	Source    model.DataSourceKind
	EventKind string
}

// DepIndex maps data streams to affected rule IDs
type DepIndex struct {
	mu    sync.RWMutex
	index map[DepKey]map[string]struct{}
}

func NewDepIndex() *DepIndex {
	return &DepIndex{
		index: make(map[DepKey]map[string]struct{}),
	}
}

func (i *DepIndex) Register(ruleId string, deps []model.Dependency) {
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, dep := range deps {
		key := DepKey{
			CameraId:  dep.CameraId,
			ROIId:     dep.ROIId,
			Source:    dep.Source,
			EventKind: dep.EventKind,
		}

		if i.index[key] == nil {
			i.index[key] = make(map[string]struct{})
		}
		i.index[key][ruleId] = struct{}{}
	}
}

func (i *DepIndex) Unregister(ruleId string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	for key, ruleIds := range i.index {
		delete(ruleIds, ruleId)
		if len(ruleIds) == 0 {
			delete(i.index, key)
		}
	}
}

func (i *DepIndex) Lookup(key DepKey) []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var result []string
	if ruleIds, ok := i.index[key]; ok {
		for id := range ruleIds {
			result = append(result, id)
		}
	}
	return result
}

func (i *DepIndex) LookupState(cameraId, roiId string) []string {
	return i.Lookup(DepKey{
		CameraId: cameraId,
		ROIId:    roiId,
		Source:   model.SourceState,
	})
}

func (i *DepIndex) LookupEvent(cameraId, eventKind string) []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	ruleIds := make(map[string]struct{})

	keys := []DepKey{
		{CameraId: cameraId, Source: model.SourceEvent, EventKind: eventKind},
		{CameraId: cameraId, Source: model.SourceEvent, EventKind: ""},
	}
	if eventKind != "*" {
		keys = append(keys, DepKey{CameraId: cameraId, Source: model.SourceEvent, EventKind: "*"})
	}

	for _, key := range keys {
		if ids, ok := i.index[key]; ok {
			for id := range ids {
				ruleIds[id] = struct{}{}
			}
		}
	}

	var result []string
	for id := range ruleIds {
		result = append(result, id)
	}
	return result
}
