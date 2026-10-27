package main

import "github.com/microsoft/durabletask-go/task"

const (
	counterName  = "go-sample-entities-counter"
	workflowName = "go-sample-entities-workflow"
)

func newRegistry() (*task.TaskRegistry, error) {
	registry := task.NewTaskRegistry()
	if err := registry.AddEntityN(counterName, counter); err != nil {
		return nil, err
	}
	if err := registry.AddOrchestratorN(workflowName, counterWorkflow); err != nil {
		return nil, err
	}
	return registry, nil
}
