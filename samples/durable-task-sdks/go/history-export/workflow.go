package main

import "github.com/microsoft/durabletask-go/task"

const orchestratorName = "GoHistoryExportOrchestrator"

func squareOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var n int
	if err := ctx.GetInput(&n); err != nil {
		return nil, err
	}
	var result int
	if err := ctx.CallActivity(squareName, task.WithActivityInput(n)).Await(&result); err != nil {
		return nil, err
	}
	return result, nil
}
