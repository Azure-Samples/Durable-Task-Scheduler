package main

import (
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

func fanOutFanIn(ctx *task.OrchestrationContext) (any, error) {
	var items []int64
	if err := ctx.GetInput(&items); err != nil {
		return nil, err
	}
	if len(items) > maxItems {
		return nil, fmt.Errorf("batch contains more than %d items", maxItems)
	}
	ctx.Logger().Info("Fanning out work", "items", len(items))
	pending := make([]task.Task, len(items))
	for i, item := range items {
		pending[i] = ctx.CallActivity(processName, task.WithActivityInput(item))
	}
	// Schedule the whole batch before waiting; also drain siblings when one fails.
	if err := ctx.WhenAll(pending...); err != nil {
		return nil, fmt.Errorf("process batch: %w", err)
	}
	results := make([]WorkResult, len(pending))
	for i, work := range pending {
		if err := work.Await(&results[i]); err != nil {
			return nil, fmt.Errorf("decode item %d: %w", i, err)
		}
	}
	var summary Summary
	if err := ctx.CallActivity(aggregateName, task.WithActivityInput(results)).Await(&summary); err != nil {
		return nil, fmt.Errorf("aggregate batch: %w", err)
	}
	return summary, nil
}
