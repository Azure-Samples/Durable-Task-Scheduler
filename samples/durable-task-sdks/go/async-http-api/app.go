package main

import (
	"context"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

func run(ctx context.Context) error {
	if *serve {
		if _, err := loopbackAddress(*listen); err != nil {
			return err
		}
	}
	registry, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, registry, func(ctx context.Context, client *dts.Client) error {
		handler := newHandler(schedulerStore{client})
		if *serve {
			return serveHTTP(ctx, *listen, handler)
		}
		return demo(ctx, handler)
	})
}

func newRegistry() (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(orchestratorName, orchestrate); err != nil {
		return nil, err
	}
	if err := registry.AddActivityN(activityName, processActivity); err != nil {
		return nil, err
	}
	return registry, nil
}
