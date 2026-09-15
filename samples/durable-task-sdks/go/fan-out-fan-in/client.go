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
		items := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("fan-out-fan-in")), api.WithInput(items))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		var summary Summary
		if err := sample.Wait(ctx, c, id, &summary); err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID api.InstanceID `json:"instance_id"`
			Summary    Summary        `json:"summary"`
		}{id, summary})
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
