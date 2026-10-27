package main

import (
	"context"
	"errors"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("function-chaining")), api.WithInput("User"))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		var output string
		if err := sample.Wait(ctx, c, id, &output); err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID api.InstanceID `json:"instance_id"`
			Output     string         `json:"output"`
		}{id, output})
	})
}

func stopOnError(c *dts.Client, id api.InstanceID, runErr *error) {
	if *runErr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := c.FetchOrchestrationMetadata(ctx, id)
	if err == nil && !state.IsComplete() {
		err = c.TerminateOrchestration(ctx, id)
		if err == nil {
			_, err = c.WaitForOrchestrationCompletion(ctx, id)
		}
	}
	*runErr = errors.Join(*runErr, err)
}
