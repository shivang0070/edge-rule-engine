package scheduler

import (
	"context"
	"sync"
	"time"
)

type Task struct {
	ID        string
	RuleID    string
	ExecuteAt int64
	Payload   any
}

type TimerQueue struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewTimerQueue() *TimerQueue {
	return &TimerQueue{
		tasks: make(map[string]*Task),
	}
}

func (q *TimerQueue) Schedule(id string, ruleId string, delay time.Duration, payload any) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks[id] = &Task{
		ID:        id,
		RuleID:    ruleId,
		ExecuteAt: time.Now().Add(delay).Unix(),
		Payload:   payload,
	}
}

func (q *TimerQueue) Cancel(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.tasks, id)
}

func (q *TimerQueue) GetDueTasks() []*Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().Unix()
	var due []*Task

	for id, task := range q.tasks {
		if task.ExecuteAt <= now {
			due = append(due, task)
			delete(q.tasks, id)
		}
	}
	return due
}

func (q *TimerQueue) Run(ctx context.Context, handler func(task *Task)) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, task := range q.GetDueTasks() {
				handler(task)
			}
		}
	}
}
