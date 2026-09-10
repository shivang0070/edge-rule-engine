package engine

import (
	"edge-rule-engine/internal/model"
	"edge-rule-engine/internal/window"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"go.uber.org/zap"
)

type Evaluator struct {
	logger *zap.Logger
}

func NewEvaluator(logger *zap.Logger) *Evaluator {
	return &Evaluator{logger: logger}
}

func (e *Evaluator) CompileExpression(expression string) (*vm.Program, error) {
	return expr.Compile(expression, expr.AsBool())
}

func (e *Evaluator) BuildStateEnv(region model.Region) map[string]any {
	return map[string]any{
		"occupancy": map[string]any{
			"person":  region.Occupancy.Person,
			"vehicle": region.Occupancy.Vehicle,
			"total":   region.Occupancy.Total,
		},
		"staffCount": region.StaffCount,
		"dwell":      buildDwellEnv(region.Precalc.Dwell),
		"entities":   region.Entities,
	}
}

func buildDwellEnv(dwell map[string]model.DwellMetrics) map[string]any {
	res := make(map[string]any)
	for k, v := range dwell {
		res[k] = map[string]any{
			"avg": v.Avg,
			"min": v.Min,
			"max": v.Max,
		}
	}
	return res
}

func (e *Evaluator) BuildEventEnv(ew *window.EventWindow) map[string]any {
	return map[string]any{
		"window": map[string]any{
			"count":         ew.Count(),
			"entranceCount": ew.CountByKind("line_crossed"),
			"exitCount":     ew.CountByKind("exit"),
		},
	}
}

func (e *Evaluator) Evaluate(program *vm.Program, env map[string]any) (bool, error) {
	result, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}
	b, ok := result.(bool)
	if !ok {
		return false, nil
	}
	return b, nil
}
