package main

import (
	"errors"

	"github.com/microsoft/durabletask-go/task"
)

func registry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	err := errors.Join(
		r.AddOrchestratorN(orderWorkflowName, orderWorkflow),
		r.AddActivityN(validateName, validateActivity),
		r.AddActivityN(chargeName, chargeActivity),
		r.AddActivityN(shipName, shipActivity),
	)
	return r, err
}
