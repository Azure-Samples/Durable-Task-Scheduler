package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoMonitoringJob"
	checkName         = "GoMonitoringCheckJobStatus"
)

type MonitorRequest struct {
	JobID                    string `json:"job_id"`
	PollIntervalMilliseconds int64  `json:"poll_interval_milliseconds"`
	TimeoutMilliseconds      int64  `json:"timeout_milliseconds"`
	CompleteAfterChecks      int    `json:"complete_after_checks"`
}

func (request MonitorRequest) validate() error {
	const dayMilliseconds = int64(24 * time.Hour / time.Millisecond)
	if strings.TrimSpace(request.JobID) == "" {
		return errors.New("job ID must not be empty")
	}
	if request.PollIntervalMilliseconds <= 0 || request.PollIntervalMilliseconds > dayMilliseconds ||
		request.TimeoutMilliseconds <= 0 || request.TimeoutMilliseconds > dayMilliseconds {
		return errors.New("poll interval and timeout must be positive and at most one day")
	}
	if request.CompleteAfterChecks < 0 || request.CompleteAfterChecks > 100 {
		return errors.New("fixture completion count must be between 0 (never) and 100")
	}
	return nil
}

type CheckInput struct {
	JobID               string `json:"job_id"`
	CheckCount          int    `json:"check_count"`
	CompleteAfterChecks int    `json:"complete_after_checks"`
}

type JobStatus struct {
	JobID         string    `json:"job_id"`
	Status        string    `json:"status"`
	CheckCount    int       `json:"check_count"`
	LastCheckTime time.Time `json:"last_check_time"`
}

type MonitorResult struct {
	JobID                          string `json:"job_id"`
	FinalStatus                    string `json:"final_status"`
	ChecksPerformed                int    `json:"checks_performed"`
	MonitoringDurationMilliseconds int64  `json:"monitoring_duration_milliseconds"`
}

func checkJobStatus(ctx task.ActivityContext) (any, error) {
	var input CheckInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.JobID == "" || input.CheckCount < 0 || input.CompleteAfterChecks < 0 {
		return nil, errors.New("invalid job status request")
	}
	// Simulation only: replace this deterministic fixture with an external status API.
	status := JobStatus{JobID: input.JobID, Status: "Running", CheckCount: input.CheckCount + 1}
	if input.CompleteAfterChecks > 0 && status.CheckCount >= input.CompleteAfterChecks {
		status.Status = "Completed"
	}
	return status, nil
}

func nextPollDelay(now, deadline time.Time, interval time.Duration) time.Duration {
	remaining := deadline.Sub(now)
	if remaining <= 0 {
		return 0
	}
	return min(interval, remaining)
}

func finishMonitoring(ctx *task.OrchestrationContext, started time.Time, status JobStatus) (any, error) {
	if err := ctx.SetCustomStatusValue(status); err != nil {
		return nil, err
	}
	return MonitorResult{
		JobID: status.JobID, FinalStatus: status.Status, ChecksPerformed: status.CheckCount,
		MonitoringDurationMilliseconds: ctx.CurrentTimeUtc.Sub(started).Milliseconds(),
	}, nil
}

func monitoringJob(ctx *task.OrchestrationContext) (any, error) {
	var request MonitorRequest
	if err := ctx.GetInput(&request); err != nil {
		return nil, err
	}
	if err := request.validate(); err != nil {
		return nil, err
	}
	started := ctx.CurrentTimeUtc
	deadline := started.Add(time.Duration(request.TimeoutMilliseconds) * time.Millisecond)
	interval := time.Duration(request.PollIntervalMilliseconds) * time.Millisecond
	status := JobStatus{JobID: request.JobID, Status: "Unknown"}

	for {
		// Always do the initial check, but never start another check at/after expiry.
		if status.CheckCount > 0 && !ctx.CurrentTimeUtc.Before(deadline) {
			status.Status = "Timeout"
			return finishMonitoring(ctx, started, status)
		}
		previousCount := status.CheckCount
		if err := ctx.CallActivity(checkName, task.WithActivityInput(CheckInput{
			JobID: request.JobID, CheckCount: previousCount, CompleteAfterChecks: request.CompleteAfterChecks,
		})).Await(&status); err != nil {
			return nil, fmt.Errorf("check job status: %w", err)
		}
		if status.JobID != request.JobID || status.CheckCount != previousCount+1 ||
			(status.Status != "Running" && status.Status != "Completed") {
			return nil, fmt.Errorf("invalid job status response: %+v", status)
		}
		status.LastCheckTime = ctx.CurrentTimeUtc
		if status.Status == "Completed" {
			return finishMonitoring(ctx, started, status)
		}
		if err := ctx.SetCustomStatusValue(status); err != nil {
			return nil, err
		}
		delay := nextPollDelay(ctx.CurrentTimeUtc, deadline, interval)
		if delay == 0 {
			status.Status = "Timeout"
			return finishMonitoring(ctx, started, status)
		}
		if err := ctx.CreateTimer(delay).Await(nil); err != nil {
			return nil, fmt.Errorf("wait for next status check: %w", err)
		}
	}
}

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, monitoringJob),
		r.AddActivityN(checkName, checkJobStatus),
	)
}

func verifyMonitor(ctx context.Context, c *dts.Client, request MonitorRequest, wantStatus string, wantChecks int) (err error) {
	id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
		api.WithInstanceID(sample.ID("monitoring")), api.WithInput(request))
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
			if err := sample.PrintJSON(lastStatus); err != nil {
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
	if err := sample.Require(result.JobID == request.JobID && result.FinalStatus == wantStatus &&
		result.ChecksPerformed == wantChecks,
		"monitor result = %+v; want job %s, %s, %d checks", result, request.JobID, wantStatus, wantChecks); err != nil {
		return err
	}
	if err := sample.Require(lastStatus.JobID == request.JobID && lastStatus.Status == wantStatus &&
		lastStatus.CheckCount == wantChecks && !lastStatus.LastCheckTime.IsZero(),
		"final custom status does not match output: %+v", lastStatus); err != nil {
		return err
	}
	if err := sample.Require(result.MonitoringDurationMilliseconds >= 0 &&
		(wantStatus != "Timeout" || result.MonitoringDurationMilliseconds >= request.TimeoutMilliseconds),
		"invalid durable monitoring duration: %+v", result); err != nil {
		return err
	}
	return sample.PrintJSON(struct {
		InstanceID api.InstanceID `json:"instance_id"`
		Result     MonitorResult  `json:"result"`
	}{id, result})
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) error {
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

func main() {
	sample.Main("monitoring", run)
}
