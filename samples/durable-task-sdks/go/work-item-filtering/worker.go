package main

import "github.com/microsoft/durabletask-go/task"

const (
	greetingName = "go-sample-filtering-greeting"
	helloName    = "go-sample-filtering-hello"
	mathName     = "go-sample-filtering-math"
	addName      = "go-sample-filtering-add"
)

func newRegistries() (*task.TaskRegistry, *task.TaskRegistry, error) {
	a, b := task.NewTaskRegistry(), task.NewTaskRegistry()
	if err := a.AddOrchestratorN(greetingName, greetingWorkflow); err != nil {
		return nil, nil, err
	}
	if err := a.AddActivityN(helloName, sayHello); err != nil {
		return nil, nil, err
	}
	if err := b.AddOrchestratorN(mathName, mathWorkflow); err != nil {
		return nil, nil, err
	}
	if err := b.AddActivityN(addName, addNumbers); err != nil {
		return nil, nil, err
	}
	return a, b, nil
}
