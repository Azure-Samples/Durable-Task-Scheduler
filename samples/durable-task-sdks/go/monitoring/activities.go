package main

import (
	"errors"
	"time"

	"github.com/microsoft/durabletask-go/task"
)

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

func checkJobStatus(ctx task.ActivityContext) (any, error) {
	var input CheckInput
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if input.JobID == "" || input.CheckCount < 0 || input.CompleteAfterChecks < 0 {
		return nil, errors.New("invalid job status request")
	}
	// Simulation only: replace this fixture with an external job-status API.
	status := JobStatus{JobID: input.JobID, Status: "Running", CheckCount: input.CheckCount + 1}
	if input.CompleteAfterChecks > 0 && status.CheckCount >= input.CompleteAfterChecks {
		status.Status = "Completed"
	}
	return status, nil
}
