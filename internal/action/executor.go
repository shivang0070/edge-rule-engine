package action

import (
	"context"
	"edge-rule-engine/internal/model"
)

type Executor interface {
	Execute(ctx context.Context, record model.ActionRecord) error
}
