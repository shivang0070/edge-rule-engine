package engine

import (
	"edge-rule-engine/internal/observation"
	"edge-rule-engine/internal/fact"
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

func (e *Evaluator) BuildRuleContext(rule *ActiveRule, obsStore *observation.Store, facts *fact.FactStore) (map[string]any, bool) {
	env := make(map[string]any)
	allFresh := true

	for _, dep := range rule.Rule.Dependencies {
		obs, freshness := obsStore.Get(dep.CameraId, dep.ROIId)
		
		if freshness == observation.Missing {
			env[dep.Alias] = nil
			allFresh = false
		} else {
			if freshness == observation.Stale {
				allFresh = false
			}
			
			// For backwards compatibility, if it's the "default" alias, we also spread its keys to root level
			obsEnv := obs.ToEnv()
			env[dep.Alias] = obsEnv
			
			if dep.Alias == "default" {
				for k, v := range obsEnv {
					env[k] = v
				}
			}
		}
	}

	return env, allFresh
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
