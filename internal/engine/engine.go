package engine

import (
	"context"
	"sync"
	"fmt"
	"time"

	"edge-rule-engine/internal/graph"
	"edge-rule-engine/internal/model"
	"edge-rule-engine/internal/store"
	"edge-rule-engine/internal/window"
	"edge-rule-engine/internal/observation"
	"edge-rule-engine/internal/fact"
	"edge-rule-engine/internal/entity"

	"go.uber.org/zap"
)

type ActionExecutor interface {
	Execute(ctx context.Context, record model.ActionRecord) error
}

type Engine struct {
	rules        map[string]*ActiveRule
	depIndex     *graph.DepIndex
	windows      *window.Manager
	observations *observation.Store
	facts        *fact.FactStore
	entities     *entity.Store
	triggers     *TriggerManager
	evaluator    *Evaluator
	executor     ActionExecutor
	execStore    store.ExecutionStore
	sseBroker    interface{ Broadcast(event string, data any) }
	logger       *zap.Logger
	mu           sync.RWMutex
}

func NewEngine(executor ActionExecutor, execStore store.ExecutionStore, sseBroker interface{ Broadcast(event string, data any) }, logger *zap.Logger) *Engine {
	return &Engine{
		rules:        make(map[string]*ActiveRule),
		depIndex:     graph.NewDepIndex(),
		windows:      window.NewManager(),
		observations: observation.NewStore(context.Background(), 30 * time.Second),
		facts:        fact.NewFactStore(),
		entities:     entity.NewStore(context.Background(), 24 * time.Hour), // entities TTL
		triggers:     NewTriggerManager(logger),
		evaluator: NewEvaluator(logger),
		executor:  executor,
		execStore: execStore,
		sseBroker: sseBroker,
		logger:    logger,
	}
}

func (e *Engine) RegisterRule(rule model.Rule) error {
	if !rule.Enabled {
		return nil
	}

	rule.ConvertV1()
	prog, err := e.evaluator.CompileExpression(rule.Condition.Expression)
	if err != nil {
		e.logger.Error("failed to compile rule expression", zap.String("ruleId", rule.ID), zap.Error(err))
		return err
	}

	active := &ActiveRule{
		Rule:    rule,
		Program: prog,
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules[rule.ID] = active
	e.depIndex.Register(rule.ID, rule.Dependencies)
	return nil
}

func (e *Engine) UnregisterRule(ruleId string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.rules, ruleId)
	e.depIndex.Unregister(ruleId)
	if e.windows != nil {
		e.windows.RemoveWindows(ruleId)
	}
	e.triggers.Remove(ruleId)
	return nil
}

func (e *Engine) UpdateRule(rule model.Rule) error {
	e.UnregisterRule(rule.ID)
	return e.RegisterRule(rule)
}

func (e *Engine) ActiveRuleCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.rules)
}

func (e *Engine) GetActiveRule(ruleId string) *ActiveRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rules[ruleId]
}

func (e *Engine) ProcessState(ctx context.Context, state model.StatePayload) {
	e.logger.Debug("received state payload", zap.String("camera", state.Camera), zap.Int("regions", len(state.Regions)))
	for _, region := range state.Regions {
		e.logger.Debug("checking region", zap.String("camera", state.Camera), zap.String("roi", region.ROIId))
		
		ruleIds := e.depIndex.LookupState(state.Camera, region.ROIId)
		wildcardIds := e.depIndex.LookupState(state.Camera, "")
		
		// deduplicate rule IDs
		ruleMap := make(map[string]struct{})
		for _, id := range ruleIds {
			ruleMap[id] = struct{}{}
		}
		for _, id := range wildcardIds {
			ruleMap[id] = struct{}{}
		}

		var rules []*ActiveRule
		e.mu.RLock()
		for id := range ruleMap {
			if rule, ok := e.rules[id]; ok {
				rules = append(rules, rule)
			}
		}
		e.mu.RUnlock()

		// 1. Update observation store and entities
		e.observations.Update(state.Camera, region.ROIId, region, state.EmittedAt)
		e.entities.UpdateFromState(state.Camera, region.ROIId, region.Entities, state.EmittedAt)

		if len(rules) > 0 {
			e.logger.Debug("found state rules for region", zap.String("camera", state.Camera), zap.String("roi", region.ROIId), zap.Int("ruleCount", len(rules)))
		}
		
		for _, rule := range rules {
			// V1 window compatibility maintenance
			if rule.Rule.Window != nil {
				stateWindow := e.windows.GetOrCreateStateWindow(rule.Rule.ID, rule.Rule.Window.DurationSeconds)
				entry := window.StateEntry{
					EmittedAt:  state.EmittedAt,
					CameraId:   state.Camera,
					ROIId:      region.ROIId,
					Occupancy:  window.OccupancyData{
						Total:   region.Occupancy.Total,
						Person:  region.Occupancy.Person,
						Vehicle: region.Occupancy.Vehicle,
					},
					StaffCount: region.StaffCount,
				}
				
				if region.Precalc.Dwell != nil {
					entry.Dwell = make(map[string]window.DwellData)
					for k, v := range region.Precalc.Dwell {
						entry.Dwell[k] = window.DwellData{Avg: v.Avg, Min: v.Min, Max: v.Max}
					}
				}
				
				if region.Entities != nil {
					for _, ent := range region.Entities {
						dw := 0.0
						if ent.DwellTime != nil {
							dw = *ent.DwellTime
						}
						entry.Entities = append(entry.Entities, window.EntityData{
							EntityType: ent.EntityType,
							TrackId:    ent.TrackId,
							DwellTime:  dw,
						})
					}
				}
				stateWindow.Add(entry)
			}

			// 2. Evaluate Rule
			e.evaluateRule(ctx, rule, state.EmittedAt, state.Camera, region.ROIId)
		}
	}
}

func (e *Engine) ProcessEvent(ctx context.Context, event model.EventPayload) {
	e.logger.Debug("received event payload", zap.String("camera", event.Camera), zap.String("kind", event.Kind), zap.String("roiId", event.ROIId))
	
	ruleIds := e.depIndex.LookupEvent(event.Camera, event.Kind)
	var rules []*ActiveRule
	e.mu.RLock()
	for _, id := range ruleIds {
		if rule, ok := e.rules[id]; ok {
			rules = append(rules, rule)
		}
	}
	e.mu.RUnlock()

	if len(rules) > 0 {
		e.logger.Debug("found event rules", zap.String("camera", event.Camera), zap.String("kind", event.Kind), zap.Int("ruleCount", len(rules)))
	}

	for _, rule := range rules {
		if rule.Rule.Window != nil {
			eventWindow := e.windows.GetOrCreateEventWindow(rule.Rule.ID, rule.Rule.Window.DurationSeconds)
			entry := window.EventEntry{
				EventId:   event.EventId,
				Kind:      event.Kind,
				EmittedAt: event.EmittedAt,
				CameraId:  event.Camera,
				ROIId:     event.ROIId,
			}
			if event.Entity != nil {
				entry.TrackId = event.Entity.TrackId
				entry.EntityType = event.Entity.EntityType
			}
			eventWindow.Add(entry)
		}

		e.evaluateRule(ctx, rule, event.EmittedAt, event.Camera, event.ROIId)
	}
}

func (e *Engine) executeAction(ctx context.Context, rule *ActiveRule, emittedAt int64, cameraId, roiId string) {
	record := model.ActionRecord{
		RuleID:      rule.Rule.ID,
		RuleName:    rule.Rule.Name,
		TriggeredAt: emittedAt,
		ActionType:  rule.Rule.Action.Type,
		CameraId:    cameraId,
		ROIId:       roiId,
		Context: map[string]any{
			"source": rule.Rule.Source,
		},
	}

	go func(execRecord model.ActionRecord) {
		bgCtx := context.Background()
		execErr := e.executor.Execute(bgCtx, execRecord)
		
		status := "success"
		errMsg := ""
		if execErr != nil {
			status = "failed"
			errMsg = execErr.Error()
			e.logger.Error("action execution failed", zap.String("ruleId", rule.Rule.ID), zap.Error(execErr))
		}

		exec := &model.RuleExecution{
			RuleID:      rule.Rule.ID,
			TriggeredAt: emittedAt,
			Status:      status,
			ActionType:  rule.Rule.Action.Type,
			Context:     fmt.Sprintf("source:%s", rule.Rule.Source),
			Error:       errMsg,
		}

		if err := e.execStore.Record(exec); err != nil {
			e.logger.Error("failed to record execution", zap.String("ruleId", rule.Rule.ID), zap.Error(err))
		}

		if e.sseBroker != nil {
			e.sseBroker.Broadcast("execution", exec)
		}
	}(record)
}

func (e *Engine) evaluateRule(ctx context.Context, rule *ActiveRule, emittedAt int64, cameraId, roiId string) {
	env, allFresh := e.evaluator.BuildRuleContext(rule, e.observations, e.facts)
	
	// Fast path for V1 compat window logic
	if rule.Rule.Window != nil {
		if rule.Rule.Source == "event" {
			eventWindow := e.windows.GetOrCreateEventWindow(rule.Rule.ID, rule.Rule.Window.DurationSeconds)
			env["window"] = map[string]any{
				"count":         eventWindow.Count(),
				"entranceCount": eventWindow.CountByKind("line_crossed"),
				"exitCount":     eventWindow.CountByKind("exit"),
			}
		} else {
			stateWindow := e.windows.GetOrCreateStateWindow(rule.Rule.ID, rule.Rule.Window.DurationSeconds)
			if env["default"] != nil {
				env["window"] = map[string]any{
					"uniquePersons":  len(stateWindow.UniqueTrackIds("person")),
					"uniqueVehicles": len(stateWindow.UniqueTrackIds("vehicle")),
				}
			}
		}
	}

	result := TriValUnknown
	if allFresh {
		res, err := e.evaluator.Evaluate(rule.Program, env)
		if err != nil {
			e.logger.Error("rule evaluation error", zap.String("ruleId", rule.Rule.ID), zap.Error(err))
			result = TriValUnknown
		} else if res {
			result = TriValTrue
		} else {
			result = TriValFalse
		}
	} else {
		// V2 could evaluate with Stale Data if configured.
		// For now, if not all fresh, we fall back to False to match V1 behavior in missing data cases.
		result = TriValFalse
	}

	conditionResult := result == TriValTrue

	if e.triggers.ShouldTrigger(rule.Rule.ID, conditionResult, emittedAt, rule.Rule.Trigger, rule.Rule.Sustain) {
		e.executeAction(ctx, rule, emittedAt, cameraId, roiId)
	}
}
