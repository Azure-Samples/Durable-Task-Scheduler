package main

import (
	"context"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/task"
	"go.opentelemetry.io/otel/trace"
)

func startWorker(ctx context.Context, tracer trace.Tracer, notificationURL string) (*sample.Host, error) {
	registry := task.NewTaskRegistry()
	if err := registerWorkflow(registry, tracer, notificationURL); err != nil {
		return nil, err
	}
	return sample.Start(ctx, registry, nil)
}

func registerWorkflow(registry *task.TaskRegistry, tracer trace.Tracer, notificationURL string) error {
	if err := registry.AddOrchestratorN(orchestratorName, orderProcessingOrchestrator); err != nil {
		return err
	}
	for _, step := range []struct {
		name, span string
		activity   task.Activity
	}{
		{validateOrderName, "app.validate_order", validateOrder},
		{processPaymentName, "app.process_payment", processPayment},
		{shipOrderName, "app.ship_order", shipOrder},
		{sendNotificationName, "app.send_notification", sendNotification(tracer, notificationURL)},
	} {
		if err := registry.AddActivityN(step.name, traceActivity(tracer, step.name, step.span, step.activity)); err != nil {
			return err
		}
	}
	return nil
}
