package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestratorName = "GoAsyncHTTPAPI"
	activityName     = "GoAsyncHTTPProcessOperation"
)

func validateProcessingTime(seconds int) error {
	if seconds < 1 || seconds > 30 {
		return errors.New("processing_time must be an integer between 1 and 30 seconds")
	}
	return nil
}

func orchestrate(ctx *task.OrchestrationContext) (any, error) {
	var input operationInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := validateProcessingTime(input.ProcessingTime); err != nil {
		return nil, err
	}
	if input.OperationID != string(ctx.ID) {
		return nil, errors.New("operation ID must match the orchestration instance ID")
	}
	var result operationResult
	if err := ctx.CallActivity(activityName, task.WithActivityInput(input)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func processActivity(ctx task.ActivityContext) (any, error) {
	var input operationInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	return processOperation(ctx.Context(), input)
}

func processOperation(ctx context.Context, input operationInput) (operationResult, error) {
	if err := validateProcessingTime(input.ProcessingTime); err != nil {
		return operationResult{}, err
	}
	// Simulated external work belongs in an activity, not in replayed orchestration code.
	timer := time.NewTimer(time.Duration(input.ProcessingTime) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return operationResult{}, ctx.Err()
	case <-timer.C:
	}
	return operationResult{
		OperationID: input.OperationID,
		Status:      "completed",
		Result:      fmt.Sprintf("Operation %s completed successfully", input.OperationID),
		ProcessedAt: float64(time.Now().UnixMilli()) / 1000,
	}, nil
}
