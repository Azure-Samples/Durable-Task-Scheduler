package main

import (
	"context"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/task"
)

const (
	workflowName = "go-sample-management-batch"
	activityName = "go-sample-management-process"
)

func startWorker(ctx context.Context) (*sample.Host, error) {
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(workflowName, batchWorkflow); err != nil {
		return nil, err
	}
	if err := registry.AddActivityN(activityName, processBatch); err != nil {
		return nil, err
	}
	return sample.StartWithWorkerContext(ctx, context.WithoutCancel(ctx), registry, nil)
}
