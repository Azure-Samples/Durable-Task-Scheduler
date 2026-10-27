package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/microsoft/durabletask-go/api"
	"github.com/microsoft/durabletask-go/task"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type activityInput struct {
	ctx   context.Context
	value any
}

func (a activityInput) Context() context.Context { return a.ctx }
func (a activityInput) GetInput(target any) error {
	body, err := json.Marshal(a.value)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func TestApplicationTraceAndOutboundPropagation(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	memory := tracetest.NewInMemoryExporter()
	telemetry, err := configureTracing(context.Background(), sdktrace.WithSyncer(memory))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := telemetry.Close(); err != nil {
			t.Error(err)
		}
	})
	tracer := newObservingTracer(telemetry.provider.Tracer("test"))
	_, caller := tracer.Start(context.Background(), callerSpanName, trace.WithSpanKind(trace.SpanKindClient))
	callerContext := caller.SpanContext()
	caller.End()
	target := notificationServer(tracer)
	defer target.Close()
	remote := callerContext.WithRemote(true)
	value := "Order-12345"
	activities := []task.Activity{validateOrder, processPayment, shipOrder, sendNotification(tracer, target.URL)}
	for i, step := range orderSteps {
		activity := traceActivity(tracer, step.Activity, step.Span, activities[i])
		want := step.Result + "(" + value + ")"
		output, err := activity(activityInput{
			ctx: trace.ContextWithRemoteSpanContext(context.Background(), remote), value: value,
		})
		if err != nil {
			t.Fatal(err)
		}
		value = output.(string)
		if value != want {
			t.Fatalf("%s returned %q, want %q", step.Activity, value, want)
		}
	}
	if err := telemetry.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if value != expectedOrderResult("Order-12345") {
		t.Fatalf("unexpected order result: %s", value)
	}
	spans := memory.GetSpans()
	if err := verifyApplicationSpans(spans, callerContext, tracer.snapshot()); err != nil {
		t.Fatal(err)
	}
	if len(spans) != 7 {
		t.Fatalf("expected only seven explicit application spans, got %d", len(spans))
	}
	observed := tracer.snapshot()
	observed[orderSteps[0].Span][0].Parent = trace.SpanContext{}
	if err := verifyApplicationSpans(spans, callerContext, observed); err == nil {
		t.Fatal("missing remote activity context accepted")
	}
	if err := verifyApplicationSpans(nil, callerContext, tracer.snapshot()); err == nil {
		t.Fatal("missing exported spans accepted")
	}
}

func TestVerificationRejectsMissingUnsampledAndRecordingParents(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	telemetry, err := configureTracing(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := telemetry.Close(); err != nil {
			t.Error(err)
		}
	}()
	tracer := telemetry.provider.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "recording")
	defer span.End()
	unsampled := span.SpanContext().WithRemote(true).WithTraceFlags(0)
	for _, parent := range []context.Context{
		context.Background(), ctx, trace.ContextWithRemoteSpanContext(context.Background(), unsampled),
	} {
		inherited := trace.SpanFromContext(parent)
		observation := spanObservation{Parent: inherited.SpanContext(), ParentRecording: inherited.IsRecording()}
		if err := verifyRestoredContext(observation, span.SpanContext()); err == nil {
			t.Fatal("invalid activity trace parent accepted")
		}
	}
}

func TestPersistedTraceContextRequiresMatchingValidCaller(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	caller := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	})
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(trace.ContextWithSpanContext(context.Background(), caller), carrier)
	wire := &api.HistoryTraceContext{TraceParent: carrier.Get("traceparent")}
	history := &api.OrchestrationHistory{
		InstanceID: "mine", ExecutionID: "execution",
		Events: []*api.HistoryEvent{{
			Type: api.HistoryEventExecutionStarted,
			ExecutionStarted: &api.HistoryExecutionStartedEvent{
				InstanceID: "mine", Name: orchestratorName, ParentTraceContext: wire,
			},
		}},
	}
	for _, step := range orderSteps {
		history.Events = append(history.Events, &api.HistoryEvent{
			Type:          api.HistoryEventTaskScheduled,
			TaskScheduled: &api.HistoryTaskScheduledEvent{Name: step.Activity, ParentTraceContext: wire},
		})
	}
	if err := verifyHistoryTrace(history, "mine", caller); err != nil {
		t.Fatal(err)
	}
	history.Events[1].TaskScheduled.ParentTraceContext = &api.HistoryTraceContext{TraceParent: "not-w3c-" + traceID.String()}
	if err := verifyHistoryTrace(history, "mine", caller); err == nil {
		t.Fatal("substring containing a trace ID was accepted as W3C context")
	}
}

type failingExporter struct{}

var exportFailure = errors.New("export failed")
var shutdownFailure = errors.New("shutdown failed")

func (failingExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return exportFailure
}
func (failingExporter) Shutdown(context.Context) error { return shutdownFailure }

func TestExporterRemembersAsynchronousErrors(t *testing.T) {
	exporter := &checkedExporter{inner: failingExporter{}}
	if err := exporter.ExportSpans(context.Background(), nil); !errors.Is(err, exportFailure) {
		t.Fatal(err)
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	session := &tracingSession{provider: provider, remote: exporter}
	if err := session.Flush(context.Background()); !errors.Is(err, exportFailure) {
		t.Fatalf("earlier export error was lost by flush: %v", err)
	}
	if err := session.Close(); !errors.Is(err, exportFailure) || !errors.Is(err, shutdownFailure) {
		t.Fatalf("flush/shutdown errors were not surfaced: %v", err)
	}
}

func TestOTLPEndpointValidation(t *testing.T) {
	for _, endpoint := range []string{"http://localhost:4318", "https://collector.example/v1/traces"} {
		if err := validateOTLPEndpoint(endpoint); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRealOTLPHTTPExporter(t *testing.T) {
	type requestReceipt struct {
		path, contentType string
		bytes             int
		err               error
	}
	requests := make(chan requestReceipt, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
		requests <- requestReceipt{r.URL.Path, r.Header.Get("Content-Type"), len(body), err}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	telemetry, err := configureTracing(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := telemetry.Close(); err != nil {
			t.Error(err)
		}
	}()
	_, span := telemetry.provider.Tracer("test").Start(context.Background(), "real-http-export")
	span.End()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := telemetry.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case receipt := <-requests:
		if receipt.path != "/v1/traces" || receipt.contentType != "application/x-protobuf" ||
			receipt.bytes == 0 || receipt.err != nil {
			t.Fatalf("invalid OTLP request: %+v", receipt)
		}
	case <-ctx.Done():
		t.Fatal("the configured OTLP receiver did not receive a request")
	}
}

func TestInvalidOTLPEndpoints(t *testing.T) {
	for _, endpoint := range []string{
		"localhost:4318", "ftp://collector", "http://user@collector",
		"https://collector/#fragment", "https://collector/?ignored=true",
	} {
		if err := validateOTLPEndpoint(endpoint); err == nil {
			t.Fatalf("invalid endpoint %q accepted", endpoint)
		}
	}
}
