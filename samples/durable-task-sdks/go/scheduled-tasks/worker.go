package main

import (
	"context"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	reportName   = "go-sample-schedules-report"
	activityName = "go-sample-schedules-send-report"
)

func newRegistry() (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(reportName, reportWorkflow); err != nil {
		return nil, err
	}
	if err := registry.AddActivityN(activityName, sendReport); err != nil {
		return nil, err
	}
	if err := dts.RegisterScheduledTasks(registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func startWorker(ctx context.Context) (*sample.Host, error) {
	registry, err := newRegistry()
	if err != nil {
		return nil, err
	}
	// Durable deletion needs a live worker, but connection setup must remain cancellable.
	return sample.StartWithWorkerContext(ctx, context.WithoutCancel(ctx), registry, nil, dts.WithScheduledTasks())
}
