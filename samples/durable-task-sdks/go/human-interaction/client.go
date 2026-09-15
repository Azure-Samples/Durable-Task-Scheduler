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
		id := sample.ID("human-interaction")
		request := ApprovalRequest{
			RequestID: string(id), Requester: "Console User", Item: "Vacation Request", TimeoutSeconds: 10,
		}
		if _, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(id), api.WithInput(request)); err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		// Simulate the approver. DTS buffers this event if the workflow is not waiting yet.
		approved := true
		response := ApprovalResponse{
			IsApproved: &approved, Approver: "Console Approver", Comments: "Automated demo response",
		}
		if err := c.RaiseEvent(ctx, id, approvalEvent, api.WithEventPayload(response)); err != nil {
			return err
		}
		var result ApprovalResult
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		return sample.PrintJSON(result)
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
