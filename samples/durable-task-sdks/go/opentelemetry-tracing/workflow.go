package main

import "github.com/microsoft/durabletask-go/task"

const orchestratorName = "GoOpenTelemetryTracingOrderProcessing"

func orderProcessingOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var order string
	if err := ctx.GetInput(&order); err != nil {
		return nil, err
	}
	for _, activity := range []string{
		validateOrderName, processPaymentName, shipOrderName, sendNotificationName,
	} {
		if err := ctx.CallActivity(activity, task.WithActivityInput(order)).Await(&order); err != nil {
			return nil, err
		}
	}
	return order, nil
}
