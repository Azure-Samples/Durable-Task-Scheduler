package main

import "github.com/microsoft/durabletask-go/task"

func acceptOrder(ctx task.ActivityContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	return "accepted:" + input.ID, nil
}

func callPartner(ctx task.ActivityContext) (any, error) {
	var input order
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	if err := input.validate(); err != nil {
		return nil, err
	}
	// A simulation with no external side effects, safe to repeat on redelivery.
	return "partner-confirmed:" + input.ID, nil
}
