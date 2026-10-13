package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

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

type MonitorResult struct {
	JobID                          string `json:"job_id"`
	FinalStatus                    string `json:"final_status"`
	ChecksPerformed                int    `json:"checks_performed"`
	MonitoringDurationMilliseconds int64  `json:"monitoring_duration_milliseconds"`
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
