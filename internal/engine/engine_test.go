package engine

import (
	"context"
	"testing"
	"time"

	"edge-rule-engine/internal/model"
	"edge-rule-engine/internal/store"
	"go.uber.org/zap"
)

// Mock Executor
type mockExecutor struct {
	Triggers []model.ActionRecord
}

func (m *mockExecutor) Execute(ctx context.Context, record model.ActionRecord) error {
	m.Triggers = append(m.Triggers, record)
	return nil
}

// Mock Exec Store
type mockExecStore struct{
	store.ExecutionStore
}

func (m *mockExecStore) Record(execution *model.RuleExecution) error { return nil }

// Mock SSE Broker
type mockBroker struct{}

func (m *mockBroker) Broadcast(event string, data any) {}

func setupEngine() (*Engine, *mockExecutor) {
	logger := zap.NewNop()
	exec := &mockExecutor{}
	execStore := &mockExecStore{}
	broker := &mockBroker{}

	eng := NewEngine(exec, execStore, broker, logger)
	return eng, exec
}

func TestMultiCameraJoin_BothFresh(t *testing.T) {
	eng, exec := setupEngine()

	rule := model.Rule{
		ID:      "multi_cam_1",
		Name:    "Crowd in both areas",
		Enabled: true,
		Source:  "state",
		Dependencies: []model.Dependency{
			{Alias: "cam1", CameraId: "C1", Source: "state"},
			{Alias: "cam2", CameraId: "C2", Source: "state"},
		},
		Condition: model.Condition{
			Expression: "cam1.occupancy.person > 5 && cam2.occupancy.person > 5",
		},
		Action: model.ActionConfig{Type: "alert"},
	}

	err := eng.RegisterRule(rule)
	if err != nil {
		t.Fatalf("RegisterRule failed: %v", err)
	}

	now := time.Now().Unix()

	eng.ProcessState(context.Background(), model.StatePayload{
		Camera:    "C1",
		EmittedAt: now,
		Regions: []model.Region{
			{ROIId: "", Occupancy: model.Occupancy{Person: 10}},
		},
	})

	if len(exec.Triggers) > 0 {
		t.Fatalf("Rule should not trigger when C2 is missing, but got %d triggers", len(exec.Triggers))
	}

	eng.ProcessState(context.Background(), model.StatePayload{
		Camera:    "C2",
		EmittedAt: now + 1,
		Regions: []model.Region{
			{ROIId: "", Occupancy: model.Occupancy{Person: 6}},
		},
	})

	if len(exec.Triggers) != 1 {
		t.Fatalf("Rule should have triggered once since both dependencies are met, got %d", len(exec.Triggers))
	}
}
