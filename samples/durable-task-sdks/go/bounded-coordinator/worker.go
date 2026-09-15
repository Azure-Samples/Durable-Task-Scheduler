package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, coordinator),
		r.AddOrchestratorN(childName, processItem),
		r.AddActivityN(getBatchName, getNextBatch),
		r.AddActivityN(applyName, applyChange),
	)
}
