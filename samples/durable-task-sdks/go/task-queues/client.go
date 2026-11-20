package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func run(ctx context.Context) error {
	return withWorker(ctx, func(ctx context.Context, c *dts.Client) error {
		id, output, err := executeOrder(ctx, c, order{ID: "order-1001"})
		if err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID api.InstanceID `json:"instance_id"`
			Status     string         `json:"status"`
			Output     orderResult    `json:"output"`
		}{id, "completed", output})
	})
}

func executeOrder(ctx context.Context, c *dts.Client, input order) (id api.InstanceID, output orderResult, err error) {
	if err := input.validate(); err != nil {
		return "", output, err
	}
	id, err = c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(sample.ID("task-queues")),
		api.WithInput(input),
		api.WithTags(map[string]string{queueTag: "orders"}))
	if err != nil {
		return id, output, err
	}
	defer stopOnError(c, id, &err)
	if err = sample.Wait(ctx, c, id, &output); err != nil {
		return id, output, err
	}
	return id, output, checkResult(input, output)
}

func checkResult(input order, output orderResult) error {
	want := orderResult{
		OrderID: input.ID, Accepted: "accepted:" + input.ID,
		Partner: "partner-confirmed:" + input.ID, ChildPartner: "partner-confirmed:" + input.ID,
	}
	if output != want {
		return fmt.Errorf("order result = %+v, want %+v", output, want)
	}
	return nil
}

func stopOnError(c *dts.Client, id api.InstanceID, runErr *error) {
	if *runErr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := c.FetchOrchestrationMetadata(ctx, id)
	if err == nil && state != nil && !state.IsComplete() {
		err = c.TerminateOrchestration(ctx, id, api.WithRecursiveTerminate(true))
		if err == nil {
			_, err = c.WaitForOrchestrationCompletion(ctx, id)
		}
	}
	*runErr = errors.Join(*runErr, err)
}
