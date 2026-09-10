package engine

import (
	"context"
	"sync"
	"fmt"

	"edge-rule-engine/internal/model"
	"edge-rule-engine/internal/store"
	"edge-rule-engine/internal/window"

	"go.uber.org/zap"
)

type ActionExecutor interface {
	Execute(ctx context.Context, record model.ActionRecord) error
}

type Engine struct {
	rules     map[string]*ActiveRule
	index     *RuleIndex
	windows   *window.Manager
	triggers  *TriggerManager
	evaluator *Evaluator
	executor  ActionExecutor
	execStore store.ExecutionStore
	sseBroker interface{ Broadcast(event string, data any) }
	logger    *zap.Logger
	mu        sync.RWMutex
}

func NewEngine(executor ActionExecutor, execStore store.ExecutionStore, sseBroker interface{ Broadcast(event string, data any) }, logger *zap.Logger) *Engine {
	return &Engine{
		rules:     make(map[string]*ActiveRule),
		index:     NewRuleIndex(),
		windows:   window.NewManager(),
		triggers:  NewTriggerManager(logger),
		evaluator: NewEvaluator(logger),
		executor:  executor,
		execStore: execStore,
		sseBroker: sseBroker,
		logger:    logger,
	}
}

func (e *Engine) RegisterRule(rule model.Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !rule.Enabled {
		return nil
	}

	prog, err := e.evaluator.CompileExpression(rule.Condition.Expression)
	if err != nil {
		e.logger.Error("failed to compile rule expression", zap.String("ruleId", rule.ID), zap.Error(err))
		return err
	}

	active := &ActiveRule{
		Rule:    rule,
		Program: prog,
	}

	e.rules[rule.ID] = active
	e.index.Add(active)
	return nil
}

func (e *Engine) UnregisterRule(ruleId string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.rules, ruleId)
	e.index.Remove(ruleId)
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
		rules := e.index.FindStateRules(state.Camera, region.ROIId)
		// also check wildcard ROIs
		wildcardRules := e.index.FindStateRules(state.Camera, "")
		rules = append(rules, wildcardRules...)

		if len(rules) > 0 {
			e.logger.Debug("found state rules for region", zap.String("camera", state.Camera), zap.String("roi", region.ROIId), zap.Int("ruleCount", len(rules)))
		}
		
		for _, rule := range rules {
			var stateWindow *window.StateWindow
			if rule.Rule.Window != nil {
				stateWindow = e.windows.GetOrCreateStateWindow(rule.Rule.ID, rule.Rule.Window.DurationSeconds)
				entry := window.StateEntry{
					EmittedAt: state.EmittedAt,
					CameraId:  state.Camera,
					ROIId:     region.ROIId,
					Occupancy: window.OccupancyData{
						Total:   region.Occupancy.Total,
						Person:  region.Occupancy.Person,
						Vehicle: region.Occupancy.Vehicle,
					},
					StaffCount: region.StaffCount,
				}
				
				if region.Precalc.Dwell != nil {
					entry.Dwell = make(map[string]window.DwellData)
					for k, v := range region.Precalc.Dwell {
						entry.Dwell[k] = window.DwellData{
							Avg: v.Avg,
							Min: v.Min,
							Max: v.Max,
						}
					}
				}
				
				for _, ent := range region.Entities {
					ed := window.EntityData{
						EntityType: ent.EntityType,
						TrackId:    ent.TrackId,
					}
					if ent.DwellTime != nil {
						ed.DwellTime = *ent.DwellTime
					}
					entry.Entities = append(entry.Entities, ed)
				}

				if stateWindow != nil {
					stateWindow.Add(entry)
				}
			}

			env := e.evaluator.BuildStateEnv(region)
			if rule.Rule.Window != nil && stateWindow != nil {
				env["window"] = map[string]any{
					"uniquePersons":  len(stateWindow.UniqueTrackIds("person")),
					"uniqueVehicles": len(stateWindow.UniqueTrackIds("vehicle")),
				}
			}

			result, err := e.evaluator.Evaluate(rule.Program, env)
			if err != nil {
				e.logger.Error("failed to evaluate state rule", zap.String("ruleId", rule.Rule.ID), zap.Error(err))
				continue
			}

			e.logger.Debug("state rule evaluated", zap.String("ruleId", rule.Rule.ID), zap.Bool("matched", result))

			if e.triggers.ShouldTrigger(rule.Rule.ID, result, state.EmittedAt, rule.Rule.Trigger, rule.Rule.Sustain) {
				e.logger.Info("rule TRIGGERED", zap.String("ruleId", rule.Rule.ID), zap.String("source", "state"), zap.String("camera", state.Camera))
				e.executeAction(ctx, rule, state.EmittedAt, state.Camera, region.ROIId)
			}
		}
	}
}

func (e *Engine) ProcessEvent(ctx context.Context, event model.EventPayload) {
	e.logger.Debug("received event payload", zap.String("camera", event.Camera), zap.String("kind", event.Kind), zap.String("roiId", event.ROIId))
	rules := e.index.FindEventRules(event.Camera, event.Kind)
	
	if len(rules) > 0 {
		e.logger.Debug("found event rules", zap.String("camera", event.Camera), zap.String("kind", event.Kind), zap.Int("ruleCount", len(rules)))
	}

	for _, rule := range rules {
		var eventWindow *window.EventWindow
		if rule.Rule.Window != nil {
			eventWindow = e.windows.GetOrCreateEventWindow(rule.Rule.ID, rule.Rule.Window.DurationSeconds)
		} else {
			eventWindow = e.windows.GetOrCreateEventWindow(rule.Rule.ID, 0)
		}

		if eventWindow != nil {
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

		env := e.evaluator.BuildEventEnv(eventWindow)

		result, err := e.evaluator.Evaluate(rule.Program, env)
		if err != nil {
			e.logger.Error("failed to evaluate event rule", zap.String("ruleId", rule.Rule.ID), zap.Error(err))
			continue
		}

		e.logger.Debug("event rule evaluated", zap.String("ruleId", rule.Rule.ID), zap.Bool("matched", result))

		if e.triggers.ShouldTrigger(rule.Rule.ID, result, event.EmittedAt, rule.Rule.Trigger, rule.Rule.Sustain) {
			e.logger.Info("rule TRIGGERED", zap.String("ruleId", rule.Rule.ID), zap.String("source", "event"), zap.String("camera", event.Camera))
			e.executeAction(ctx, rule, event.EmittedAt, event.Camera, event.ROIId)
		}
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

	execErr := e.executor.Execute(ctx, record)
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
}
