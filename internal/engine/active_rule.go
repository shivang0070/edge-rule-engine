package engine

import (
	"edge-rule-engine/internal/model"

	"github.com/expr-lang/expr/vm"
)

type ActiveRule struct {
	Rule    model.Rule
	Program *vm.Program
}
