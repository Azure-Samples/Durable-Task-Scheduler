package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoFunctionChaining"
	sayHelloName      = "GoFunctionChainingSayHello"
	processName       = "GoFunctionChainingProcessGreeting"
	finalizeName      = "GoFunctionChainingFinalizeResponse"
)

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, functionChaining),
		r.AddActivityN(sayHelloName, sayHello),
		r.AddActivityN(processName, processGreeting),
		r.AddActivityN(finalizeName, finalizeResponse),
	)
}
