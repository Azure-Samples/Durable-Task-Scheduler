package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	r, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	err = sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
		approve, reject := true, false
		for _, scenario := range []struct {
			name     string
			decision *bool
		}{{"approve", &approve}, {"reject", &reject}, {"timeout", nil}} {
			if err := verifyRequest(ctx, c, scenario.name, scenario.decision); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func verifyRequest(ctx context.Context, c *dts.Client, scenario string, decision *bool) (err error) {
	id := sample.ID("human-interaction-" + scenario)
	request := ApprovalRequest{
		RequestID: string(id), Requester: "Console User", Item: "Vacation Request", TimeoutSeconds: 10,
	}
	if decision == nil {
		request.TimeoutSeconds = 1
	}
	if _, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(id), api.WithInput(request)); err != nil {
		return err
	}
	defer stopOnError(c, id, &err)

	want := ApprovalResult{RequestID: string(id), Status: "Timeout"}
	if decision != nil {
		if err := sample.Until(ctx, 50*time.Millisecond, func() (bool, error) {
			state, err := c.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
			if err != nil {
				return false, err
			}
			if state.IsComplete() {
				return false, fmt.Errorf("%s ended before a response: %s", id, state.RuntimeStatus)
			}
			if state.SerializedCustomStatus == "" {
				return false, nil
			}
			var status ApprovalResult
			if err := state.ReadCustomStatus(&status); err != nil {
				return false, err
			}
			return status.Status == "Pending", nil
		}); err != nil {
			return err
		}
		response := ApprovalResponse{
			IsApproved: decision, Approver: "Console Approver", Comments: "Automated demo response",
		}
		if err := c.RaiseEvent(ctx, id, approvalEvent, api.WithEventPayload(response)); err != nil {
			return err
		}
		want.Status = "Rejected"
		if *decision {
			want.Status = "Approved"
		}
		want.Approver = response.Approver
	}
	var result ApprovalResult
	if err := sample.Wait(ctx, c, id, &result); err != nil {
		return err
	}
	return testutil.Require(result == want, "%s result = %+v, want %+v", scenario, result, want)
}
