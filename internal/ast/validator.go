package ast

import (
	"errors"
	"edge-rule-engine/internal/model"
)

func ValidateRule(rule *model.Rule) error {
	if err := rule.Validate(); err != nil {
		return err
	}

	if len(rule.Patterns) > 0 {
		for _, pat := range rule.Patterns {
			if len(pat.Steps) < 1 {
				return errors.New("pattern must have at least one step")
			}
			for _, step := range pat.Steps {
				if step.DepAlias == "" {
					return errors.New("pattern step must reference a dependency alias")
				}
			}
		}
	}
	return nil
}
