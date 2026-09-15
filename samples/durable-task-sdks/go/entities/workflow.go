package main

import (
	"errors"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

const resetDelay = 5 * time.Second

type workflowResult struct {
	Before  int       `json:"before"`
	After   int       `json:"after"`
	ReadAt  time.Time `json:"read_at"`
	DueAt   time.Time `json:"due_at"`
	ResetAt time.Time `json:"reset_at"`
}

func counterWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var id api.EntityID
	if err := ctx.GetInput(&id); err != nil {
		return nil, err
	}
	for _, operation := range []struct {
		name   string
		amount int
	}{{"add", 10}, {"add", 5}, {"subtract", 3}} {
		if err := ctx.SignalEntity(id, operation.name, task.WithSignalEntityInput(operation.amount)); err != nil {
			return nil, err
		}
	}
	var before int
	if err := ctx.CallEntity(id, "get").Await(&before); err != nil {
		return nil, err
	}
	readAt := ctx.CurrentTimeUtc
	due := readAt.Add(resetDelay)
	if err := ctx.SignalEntity(id, "reset", task.WithSignalEntityScheduledTime(due)); err != nil {
		return nil, err
	}
	if err := ctx.CreateTimer(resetDelay).Await(nil); err != nil {
		return nil, err
	}

	// A signal has no reply. Wait durably for delivery, which may lag its due time.
	for attempt := 0; attempt < 20; attempt++ {
		var state counterState
		if err := ctx.CallEntity(id, "snapshot").Await(&state); err != nil {
			return nil, err
		}
		if !state.ResetAt.IsZero() {
			return workflowResult{
				Before: before, After: state.Value,
				ReadAt: readAt, DueAt: due, ResetAt: state.ResetAt,
			}, nil
		}
		if err := ctx.CreateTimer(500 * time.Millisecond).Await(nil); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("scheduled counter reset was not delivered before the workflow deadline")
}
