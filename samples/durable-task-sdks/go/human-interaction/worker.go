package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, humanInteraction),
		r.AddActivityN(submitName, submitApprovalRequest),
		r.AddActivityN(processName, processApproval),
	)
}
