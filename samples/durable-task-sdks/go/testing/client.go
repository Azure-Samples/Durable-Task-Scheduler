package main

import (
	"context"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func run(ctx context.Context) error {
	r, err := registry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		input := order{
			Customer: "Alice",
			Items:    []item{{Name: "Widget", Quantity: 2, UnitPriceCents: 1000}},
		}
		id, err := c.ScheduleNewOrchestration(ctx, orderWorkflowName,
			api.WithInstanceID(sample.ID("testing")), api.WithInput(input))
		if err != nil {
			return err
		}
		var result orderResult
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		return sample.PrintJSON(result)
	})
}
