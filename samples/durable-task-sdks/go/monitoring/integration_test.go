package main

import (
	"context"
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
		if err := verifyMonitor(ctx, c, MonitorRequest{
			JobID: string(sample.ID("job-completes")), PollIntervalMilliseconds: 250,
			TimeoutMilliseconds: 20000, CompleteAfterChecks: 4,
		}, "Completed", 4); err != nil {
			return err
		}
		return verifyMonitor(ctx, c, MonitorRequest{
			JobID: string(sample.ID("job-times-out")), PollIntervalMilliseconds: 2000,
			TimeoutMilliseconds: 1000, CompleteAfterChecks: 0,
		}, "Timeout", 1)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func verifyMonitor(ctx context.Context, c *dts.Client, request MonitorRequest, wantStatus string, wantChecks int) (err error) {
	id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(sample.ID("monitoring-test")), api.WithInput(request))
	if err != nil {
		return err
	}
	defer stopOnError(c, id, &err)

	var lastSerializedStatus string
	var lastStatus JobStatus
	if err := sample.Until(ctx, 100*time.Millisecond, func() (bool, error) {
		state, err := c.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
		if err != nil {
			return false, err
		}
		if state.SerializedCustomStatus != "" && state.SerializedCustomStatus != lastSerializedStatus {
			if err := state.ReadCustomStatus(&lastStatus); err != nil {
				return false, err
			}
			lastSerializedStatus = state.SerializedCustomStatus
		}
		return state.IsComplete(), nil
	}); err != nil {
		return err
	}
	var result MonitorResult
	if err := sample.Wait(ctx, c, id, &result); err != nil {
		return err
	}
	if err := testutil.Require(result.JobID == request.JobID && result.FinalStatus == wantStatus &&
		result.ChecksPerformed == wantChecks,
		"monitor result = %+v; want job %s, %s, %d checks", result, request.JobID, wantStatus, wantChecks); err != nil {
		return err
	}
	if err := testutil.Require(lastStatus.JobID == request.JobID && lastStatus.Status == wantStatus &&
		lastStatus.CheckCount == wantChecks && !lastStatus.LastCheckTime.IsZero(),
		"final custom status does not match output: %+v", lastStatus); err != nil {
		return err
	}
	return testutil.Require(result.MonitoringDurationMilliseconds >= 0 &&
		(wantStatus != "Timeout" || result.MonitoringDurationMilliseconds >= request.TimeoutMilliseconds),
		"invalid durable monitoring duration: %+v", result)
}
