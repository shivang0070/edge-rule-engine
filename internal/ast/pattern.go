package ast

import (
	"edge-rule-engine/internal/model"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type PatternEvaluator struct {
	Steps []StepEvaluator
}

type StepEvaluator struct {
	DepAlias   string
	EventKind  string
	Program    *vm.Program
}

func CompilePattern(pattern model.EventPattern) (*PatternEvaluator, error) {
	evaluator := &PatternEvaluator{}
	for _, step := range pattern.Steps {
		var prog *vm.Program
		var err error
		if step.Expression != "" {
			prog, err = expr.Compile(step.Expression, expr.AsBool())
			if err != nil {
				return nil, err
			}
		}
		
		evaluator.Steps = append(evaluator.Steps, StepEvaluator{
			DepAlias:   step.DepAlias,
			EventKind:  step.EventKind,
			Program:    prog,
		})
	}
	return evaluator, nil
}
