package main

import (
	"context"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func run(ctx context.Context) error {
	registry, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, registry, func(ctx context.Context, c *dts.Client) error {
		counterID := api.NewEntityID(counterName, string(sample.ID("entities-counter")))
		id := sample.ID("entities-workflow")
		if _, err := c.ScheduleNewOrchestration(ctx, workflowName,
			api.WithInstanceID(id), api.WithInput(counterID)); err != nil {
			return err
		}
		var result workflowResult
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		fmt.Printf("Counter before scheduled reset: %d\n", result.Before)
		fmt.Printf("Counter after scheduled reset: %d\n", result.After)
		return nil
	})
}
