package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const callerSpanName = "app.schedule_order"

type scheduledOrder struct {
	InstanceID api.InstanceID
	TraceID    string
}

func run(ctx context.Context) (err error) {
	telemetry, err := configureTracing(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, telemetry.Close()) }()
	tracer := telemetry.provider.Tracer("go-order-processing-sample")
	target := notificationServer(tracer)
	defer target.Close()
	host, err := startWorker(ctx, tracer, target.URL)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()
	fmt.Println("Processing synthetic order Order-12345")
	order, err := scheduleOrder(ctx, host.Client, tracer, "Order-12345")
	if err != nil {
		return err
	}
	var result string
	if err := sample.Wait(ctx, host.Client, order.InstanceID, &result); err != nil {
		return err
	}
	fmt.Printf("Result: %s\nTrace ID: %s\nInstance: %s\n", result, order.TraceID, order.InstanceID)
	if telemetry.remote == nil {
		fmt.Println("Set OTEL_EXPORTER_OTLP_ENDPOINT to visualize application spans.")
	}
	return nil
}

func scheduleOrder(ctx context.Context, client *dts.Client, tracer trace.Tracer, orderID string) (scheduledOrder, error) {
	id := sample.ID("tracing")
	callerCtx, caller := tracer.Start(ctx, callerSpanName, trace.WithSpanKind(trace.SpanKindClient))
	defer caller.End()
	caller.SetAttributes(attribute.String("durabletask.task.instance_id", string(id)))
	_, err := client.ScheduleNewOrchestration(callerCtx, orchestratorName,
		api.WithInstanceID(id), api.WithInput(orderID))
	if err != nil {
		caller.RecordError(err)
		caller.SetStatus(codes.Error, "schedule failed")
	}
	return scheduledOrder{InstanceID: id, TraceID: caller.SpanContext().TraceID().String()}, err
}
