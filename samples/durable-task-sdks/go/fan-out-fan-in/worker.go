package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoFanOutFanIn"
	processName       = "GoFanOutFanInProcessWorkItem"
	aggregateName     = "GoFanOutFanInAggregateResults"
)

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, fanOutFanIn),
		r.AddActivityN(processName, processWorkItem),
		r.AddActivityN(aggregateName, aggregateResults),
	)
}
