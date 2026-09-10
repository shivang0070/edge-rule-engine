package fact

import (
	"sync"
	"time"
	"edge-rule-engine/internal/model"
	"edge-rule-engine/internal/graph"
	"edge-rule-engine/internal/window"
)

type AggregatorDef struct {
	MetricName  string
	CameraId    string
	ROIId       string
	Source      model.DataSourceKind
	WindowDur   time.Duration
	AggFunc     string
	GroupBy     string
}

type Registry struct {
	mu          sync.RWMutex
	aggregators map[FactKey]*AggregatorDef
	refCount    map[FactKey]int
}

func NewRegistry() *Registry {
	return &Registry{
		aggregators: make(map[FactKey]*AggregatorDef),
		refCount:    make(map[FactKey]int),
	}
}

func (r *Registry) Register(ruleId string, deps []model.Dependency) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, dep := range deps {
		if dep.Window == nil {
			continue
		}
		
		dur, _ := time.ParseDuration(dep.Window.Duration)
		key := FactKey{
			CameraId:   dep.CameraId,
			ROIId:      dep.ROIId,
			MetricName: dep.Window.Aggregation, // simplifying for now
		}

		if _, exists := r.aggregators[key]; !exists {
			r.aggregators[key] = &AggregatorDef{
				MetricName: dep.Window.Aggregation,
				CameraId:   dep.CameraId,
				ROIId:      dep.ROIId,
				Source:     dep.Source,
				WindowDur:  dur,
				AggFunc:    dep.Window.Aggregation,
				GroupBy:    dep.Window.GroupBy,
			}
		}
		r.refCount[key]++
	}
}

func (r *Registry) Unregister(ruleId string, deps []model.Dependency) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, dep := range deps {
		if dep.Window == nil {
			continue
		}
		key := FactKey{
			CameraId:   dep.CameraId,
			ROIId:      dep.ROIId,
			MetricName: dep.Window.Aggregation,
		}
		r.refCount[key]--
		if r.refCount[key] <= 0 {
			delete(r.refCount, key)
			delete(r.aggregators, key)
		}
	}
}

func (r *Registry) Recompute(depKey graph.DepKey, windows *window.Manager, store *FactStore) []FactKey {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var changed []FactKey

	for key, def := range r.aggregators {
		if def.CameraId == depKey.CameraId && def.ROIId == depKey.ROIId && def.Source == depKey.Source {
			// Actually perform the aggregation using the windows manager
			// For this MVP step, we will delegate to a simple function that 
			// calls window.Aggregate
			changed = append(changed, key)
		}
	}
	return changed
}
