package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	orchestratorName = "GoOpenTelemetryTracingOrderProcessing"
	callerSpanName   = "app.schedule_order"
	outboundSpanName = "app.notification_http"
	serverSpanName   = "app.notification_endpoint"
)

type orderStep struct {
	Activity string
	Span     string
	Result   string
}

var orderSteps = []orderStep{
	{Activity: "GoOpenTelemetryTracingValidateOrder", Span: "app.validate_order", Result: "Validated"},
	{Activity: "GoOpenTelemetryTracingProcessPayment", Span: "app.process_payment", Result: "Paid"},
	{Activity: "GoOpenTelemetryTracingShipOrder", Span: "app.ship_order", Result: "Shipped"},
	{Activity: "GoOpenTelemetryTracingSendNotification", Span: "app.send_notification", Result: "Notified"},
}

type notificationReceipt struct {
	TraceID      string `json:"traceId"`
	ParentSpanID string `json:"parentSpanId"`
	SpanID       string `json:"spanId"`
	Sampled      bool   `json:"sampled"`
}

type stepResult struct {
	Value        string               `json:"value"`
	TraceID      string               `json:"traceId"`
	ParentSpanID string               `json:"parentSpanId"`
	SpanID       string               `json:"spanId"`
	Notification *notificationReceipt `json:"notification,omitempty"`
}

type orderResult struct {
	Value string       `json:"value"`
	Steps []stepResult `json:"steps"`
}

func main() {
	sample.Main("opentelemetry-tracing", run)
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
	registry := task.NewTaskRegistry()
	if err := registry.AddOrchestratorN(orchestratorName, orderProcessingOrchestrator); err != nil {
		return err
	}
	for i, step := range orderSteps {
		targetURL := ""
		if i == len(orderSteps)-1 {
			targetURL = target.URL
		}
		if err := registry.AddActivityN(step.Activity, tracedActivity(tracer, step, targetURL)); err != nil {
			return err
		}
	}
	host, err := sample.Start(ctx, registry, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, host.Close()) }()
	id := sample.ID("tracing")
	orderID := "Order-12345"
	callerCtx, caller := tracer.Start(ctx, callerSpanName, trace.WithSpanKind(trace.SpanKindClient))
	callerContext := caller.SpanContext()
	if !callerContext.IsValid() || !callerContext.IsSampled() {
		caller.End()
		return errors.New("caller must have a valid, sampled trace context")
	}
	caller.SetAttributes(attribute.String("durabletask.task.instance_id", string(id)))
	_, scheduleErr := host.Client.ScheduleNewOrchestration(callerCtx, orchestratorName,
		api.WithInstanceID(id), api.WithInput(orderID))
	if scheduleErr != nil {
		caller.RecordError(scheduleErr)
		caller.SetStatus(codes.Error, "schedule failed")
	}
	caller.End()
	if scheduleErr != nil {
		return scheduleErr
	}
	var result orderResult
	if err := sample.Wait(ctx, host.Client, id, &result); err != nil {
		return err
	}
	if err := verifyOrderResult(result, orderID, callerContext.TraceID().String()); err != nil {
		return err
	}
	metadata, err := host.Client.FetchOrchestrationMetadata(ctx, id)
	if err != nil {
		return err
	}
	if metadata == nil || metadata.ExecutionID == "" {
		return errors.New("completed orchestration is missing its execution ID")
	}
	history, err := host.Client.GetOrchestrationHistory(ctx, id, api.HistoryQuery{
		ExecutionID: metadata.ExecutionID, MaxEvents: 128, MaxBytes: 1024 * 1024,
	})
	if err != nil {
		return err
	}
	if err := verifyHistoryTrace(history, id, callerContext); err != nil {
		return err
	}
	if err := telemetry.Flush(ctx); err != nil {
		return err
	}
	if err := verifyApplicationSpans(telemetry.memory.GetSpans(), callerContext, result); err != nil {
		return err
	}
	fmt.Printf("Result: %s\n", result.Value)
	fmt.Printf("Trace ID: %s; instance: %s\n", callerContext.TraceID(), id)
	fmt.Println("Verified sampled caller, 4 durable activity trace contexts, 4 user activity spans, and HTTP client/server propagation")
	if telemetry.remote != nil {
		fmt.Println("Application spans also exported over OTLP/HTTP")
	} else {
		fmt.Println("Application spans verified in memory; set OTEL_EXPORTER_OTLP_ENDPOINT for Jaeger")
	}
	return nil
}

func orderProcessingOrchestrator(ctx *task.OrchestrationContext) (any, error) {
	var value string
	if err := ctx.GetInput(&value); err != nil {
		return nil, err
	}
	result := orderResult{}
	for _, step := range orderSteps {
		var output stepResult
		if err := ctx.CallActivity(step.Activity, task.WithActivityInput(value)).Await(&output); err != nil {
			return nil, err
		}
		value = output.Value
		result.Steps = append(result.Steps, output)
	}
	result.Value = value
	return result, nil
}

func tracedActivity(tracer trace.Tracer, step orderStep, targetURL string) task.Activity {
	return func(activity task.ActivityContext) (result any, err error) {
		// The Go SDK restores a NON-recording remote context. DTS, not this
		// worker, owns the durable scheduling/execution spans.
		inherited := trace.SpanFromContext(activity.Context())
		parent := inherited.SpanContext()
		if !parent.IsValid() || !parent.IsSampled() || !parent.IsRemote() || inherited.IsRecording() {
			return nil, errors.New("activity did not receive a sampled, non-recording remote DTS trace context")
		}
		ctx, span := tracer.Start(activity.Context(), step.Span, trace.WithSpanKind(trace.SpanKindInternal))
		defer func() {
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "activity failed")
			}
			span.End()
		}()
		var input string
		if err := activity.GetInput(&input); err != nil {
			return nil, err
		}
		output := stepResult{
			Value:        step.Result + "(" + input + ")",
			TraceID:      span.SpanContext().TraceID().String(),
			ParentSpanID: parent.SpanID().String(),
			SpanID:       span.SpanContext().SpanID().String(),
		}
		span.SetAttributes(attribute.String("sample.activity", step.Activity))
		if targetURL != "" {
			receipt, err := callNotification(ctx, tracer, targetURL)
			if err != nil {
				return nil, err
			}
			output.Notification = &receipt
		}
		return output, nil
	}
}

func notificationServer(tracer trace.Tracer) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		parent := trace.SpanContextFromContext(ctx)
		if !parent.IsValid() || !parent.IsSampled() || !parent.IsRemote() {
			http.Error(w, "missing sampled W3C trace context", http.StatusBadRequest)
			return
		}
		_, span := tracer.Start(ctx, serverSpanName, trace.WithSpanKind(trace.SpanKindServer))
		receipt := notificationReceipt{
			TraceID: span.SpanContext().TraceID().String(), ParentSpanID: parent.SpanID().String(),
			SpanID: span.SpanContext().SpanID().String(), Sampled: span.SpanContext().IsSampled(),
		}
		// End before replying so completion of the outbound call also guarantees
		// that the self-contained in-memory exporter has this server span.
		span.End()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(receipt); err != nil {
			return // A broken response is reported by the calling activity.
		}
	}))
}

func callNotification(ctx context.Context, tracer trace.Tracer, targetURL string) (receipt notificationReceipt, err error) {
	ctx, span := tracer.Start(ctx, outboundSpanName, trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "HTTP call failed")
		}
		span.End()
	}()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return receipt, err
	}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(request.Header))
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return receipt, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		return receipt, fmt.Errorf("notification endpoint returned %s", response.Status)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&receipt); err != nil {
		return receipt, err
	}
	if receipt.TraceID != span.SpanContext().TraceID().String() ||
		receipt.ParentSpanID != span.SpanContext().SpanID().String() || !receipt.Sampled {
		return receipt, errors.New("HTTP endpoint received an unrelated or unsampled trace context")
	}
	return receipt, nil
}
