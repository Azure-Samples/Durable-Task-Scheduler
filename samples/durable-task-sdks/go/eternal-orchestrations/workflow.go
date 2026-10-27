package main

import (
	"fmt"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoEternalPeriodicCleanup"
	cleanupName       = "GoEternalCleanupTask"
	iterations        = 5
	cleanupInterval   = 250 * time.Millisecond
)

type CleanupState struct {
	Iteration    int `json:"iteration"`
	TotalRemoved int `json:"total_removed"`
}

func (state CleanupState) validate() error {
	if state.Iteration < 1 || state.Iteration > iterations || state.TotalRemoved < 0 {
		return fmt.Errorf("invalid cleanup carry-forward state: %+v", state)
	}
	return nil
}

type CleanupResult struct {
	Iterations   int    `json:"iterations"`
	TotalRemoved int    `json:"total_removed"`
	LastMessage  string `json:"last_message"`
}

func periodicCleanup(ctx *task.OrchestrationContext) (any, error) {
	var state CleanupState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := state.validate(); err != nil {
		return nil, err
	}
	var receipt CleanupReceipt
	if err := ctx.CallActivity(cleanupName, task.WithActivityInput(state.Iteration)).Await(&receipt); err != nil {
		return nil, fmt.Errorf("cleanup cycle %d: %w", state.Iteration, err)
	}
	if receipt.Iteration != state.Iteration || receipt.Message != "Cleanup completed" {
		return nil, fmt.Errorf("invalid cleanup receipt: %+v", receipt)
	}
	state.TotalRemoved += len(receipt.Removed)
	if err := ctx.SetCustomStatusValue(state); err != nil {
		return nil, err
	}
	if err := ctx.CreateTimer(cleanupInterval).Await(nil); err != nil {
		return nil, fmt.Errorf("cleanup interval: %w", err)
	}
	if state.Iteration == iterations {
		return CleanupResult{
			Iterations: state.Iteration, TotalRemoved: state.TotalRemoved, LastMessage: receipt.Message,
		}, nil
	}

	state.Iteration++
	ctx.ContinueAsNew(state, task.WithKeepUnprocessedEvents())
	return nil, nil
}
