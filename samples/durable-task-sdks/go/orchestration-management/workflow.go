package main

import (
	"fmt"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

const (
	releaseEvent = "go-sample-management-release"
	waiting      = "waiting-for-release"
)

type batchInput struct {
	BatchID        string `json:"batch_id"`
	ItemCount      int    `json:"item_count"`
	WaitForRelease bool   `json:"wait_for_release,omitempty"`
}

type batchResult struct {
	BatchID        string `json:"batch_id"`
	ItemsProcessed int    `json:"items_processed"`
	Status         string `json:"status"`
}

func batchWorkflow(ctx *task.OrchestrationContext) (any, error) {
	var input batchInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.WaitForRelease {
		if err := ctx.SetCustomStatusValue(waiting); err != nil {
			return nil, err
		}
		var command string
		if err := ctx.WaitForSingleEvent(releaseEvent, 45*time.Second).Await(&command); err != nil {
			return nil, err
		}
		if command != "process" {
			return nil, fmt.Errorf("unexpected release command %q", command)
		}
	}
	var result batchResult
	if err := ctx.CallActivity(activityName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}
