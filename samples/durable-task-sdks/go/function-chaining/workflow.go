package main

import (
	"fmt"

	"github.com/microsoft/durabletask-go/task"
)

func functionChaining(ctx *task.OrchestrationContext) (any, error) {
	var name string
	if err := ctx.GetInput(&name); err != nil {
		return nil, err
	}
	ctx.Logger().Info("Starting greeting pipeline", "recipient", name)

	var greeting Greeting
	if err := ctx.CallActivity(sayHelloName, task.WithActivityInput(name)).Await(&greeting); err != nil {
		return nil, fmt.Errorf("create greeting: %w", err)
	}
	if err := ctx.CallActivity(processName, task.WithActivityInput(greeting)).Await(&greeting); err != nil {
		return nil, fmt.Errorf("process greeting: %w", err)
	}
	if err := ctx.CallActivity(finalizeName, task.WithActivityInput(greeting)).Await(&greeting); err != nil {
		return nil, fmt.Errorf("finalize greeting: %w", err)
	}
	return greeting.Message, nil
}
