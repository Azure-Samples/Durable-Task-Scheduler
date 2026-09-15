package main

import (
	"fmt"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoBoundedCoordinator"
	childName         = "GoBoundedCoordinatorProcessItem"
	getBatchName      = "GoBoundedCoordinatorGetNextBatch"
	applyName         = "GoBoundedCoordinatorApplyChange"
	batchLimit        = 5
)

type CoordinatorState struct {
	Cursor      string `json:"cursor"`
	BatchNumber int    `json:"batch_number"`
	Processed   int    `json:"processed"`
}

func (state CoordinatorState) validate() error {
	if state.BatchNumber < 0 || state.BatchNumber >= totalBatches ||
		state.Cursor != cursorFor(state.BatchNumber) || state.Processed != state.BatchNumber*itemsPerBatch {
		return fmt.Errorf("invalid coordinator carry-forward state: %+v", state)
	}
	return nil
}

type CoordinatorResult struct {
	TotalBatches int  `json:"total_batches"`
	Processed    int  `json:"processed"`
	Completed    bool `json:"completed"`
}

func coordinator(ctx *task.OrchestrationContext) (any, error) {
	var state CoordinatorState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := state.validate(); err != nil {
		return nil, err
	}
	var batch Batch
	if err := ctx.CallActivity(getBatchName, task.WithActivityInput(BatchRequest{
		Cursor: state.Cursor, MaxItems: batchLimit,
	})).Await(&batch); err != nil {
		return nil, fmt.Errorf("read bounded batch: %w", err)
	}
	if len(batch.Items) > batchLimit {
		return nil, fmt.Errorf("source returned %d items, exceeding the batch limit %d", len(batch.Items), batchLimit)
	}

	pending := make([]task.Task, len(batch.Items))
	for i, item := range batch.Items {
		pending[i] = ctx.CallSubOrchestrator(childName,
			task.WithSubOrchestrationInstanceID(childID(ctx.ID, item.ID)),
			task.WithSubOrchestratorInput(item))
	}
	// Finish every child before discarding this execution's history.
	if err := ctx.WhenAll(pending...); err != nil {
		return nil, fmt.Errorf("drain child batch: %w", err)
	}
	state.BatchNumber++
	state.Processed += len(batch.Items)
	state.Cursor = batch.NextCursor
	if err := ctx.SetCustomStatusValue(state); err != nil {
		return nil, err
	}
	if batch.HasMore {
		ctx.ContinueAsNew(state, task.WithKeepUnprocessedEvents())
		return nil, nil
	}
	return CoordinatorResult{TotalBatches: state.BatchNumber, Processed: state.Processed, Completed: true}, nil
}

func processItem(ctx *task.OrchestrationContext) (any, error) {
	var item Item
	if err := ctx.GetInput(&item); err != nil {
		return nil, err
	}
	var receipt string
	if err := ctx.CallActivity(applyName, task.WithActivityInput(item)).Await(&receipt); err != nil {
		return nil, fmt.Errorf("apply item %s: %w", item.ID, err)
	}
	return receipt, nil
}

func childID(parent api.InstanceID, itemID string) string {
	return string(parent) + "-" + itemID
}
