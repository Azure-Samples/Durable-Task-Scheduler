package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestActivityErrorsAreReturnedAndTraced(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	memory := tracetest.NewInMemoryExporter()
	telemetry, err := configureTracing(t.Context(), sdktrace.WithSyncer(memory))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := telemetry.Close(); err != nil {
			t.Error(err)
		}
	})
	tracer := telemetry.provider.Tracer("test")
	activity := traceActivity(tracer, validateOrderName, "app.validate_order", validateOrder)
	for _, input := range []any{"", 42} {
		if _, err := activity(activityInput{ctx: context.Background(), value: input}); err == nil {
			t.Fatalf("invalid order input accepted: %v", input)
		}
	}
	for _, span := range memory.GetSpans() {
		if span.Status.Code != codes.Error {
			t.Fatalf("activity failure was not recorded on span %s", span.Name)
		}
	}
}

func TestNotificationHTTPFailureIsNotHidden(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	telemetry, err := configureTracing(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := telemetry.Close(); err != nil {
			t.Error(err)
		}
	})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer target.Close()
	if err := callNotification(t.Context(), telemetry.provider.Tracer("test"), target.URL); err == nil {
		t.Fatal("notification service failure was ignored")
	}
}
