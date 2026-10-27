package main

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	if err := exerciseTracedOrder(ctx); err != nil {
		t.Fatal(err)
	}
}

func exerciseTracedOrder(ctx context.Context) (err error) {
	memory := tracetest.NewInMemoryExporter()
	telemetry, err := configureTracing(ctx, sdktrace.WithSyncer(memory))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, telemetry.Close()) }()
	tracer := newObservingTracer(telemetry.provider.Tracer("go-order-processing-sample"))
	target := notificationServer(tracer)
	defer target.Close()
	host, err := startWorker(ctx, tracer, target.URL)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()

	const input = "Order-12345"
	order, err := scheduleOrder(ctx, host.Client, tracer, input)
	if err != nil {
		return err
	}
	var result string
	if err := sample.Wait(ctx, host.Client, order.InstanceID, &result); err != nil {
		return err
	}
	if err := testutil.Require(result == expectedOrderResult(input), "unexpected order result: %q", result); err != nil {
		return err
	}
	metadata, err := host.Client.FetchOrchestrationMetadata(ctx, order.InstanceID)
	if err != nil {
		return err
	}
	if metadata == nil || metadata.ExecutionID == "" {
		return errors.New("completed order omitted its execution ID")
	}
	history, err := host.Client.GetOrchestrationHistory(ctx, order.InstanceID, api.HistoryQuery{
		ExecutionID: metadata.ExecutionID, MaxEvents: 128, MaxBytes: 1024 * 1024,
	})
	if err != nil {
		return err
	}
	if err := telemetry.Flush(ctx); err != nil {
		return err
	}
	spans := memory.GetSpans()
	caller, err := callerContext(spans, order.TraceID)
	if err != nil {
		return err
	}
	if err := verifyHistoryTrace(history, order.InstanceID, caller); err != nil {
		return err
	}
	if err := verifyOrderHistory(history, input); err != nil {
		return err
	}
	return verifyApplicationSpans(spans, caller, tracer.snapshot())
}
