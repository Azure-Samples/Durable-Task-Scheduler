package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/durabletask-go/task"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type tracingSession struct {
	provider *sdktrace.TracerProvider
	remote   *checkedExporter
}

func configureTracing(ctx context.Context, additional ...sdktrace.TracerProviderOption) (*tracingSession, error) {
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "GoOrderProcessingSample"))),
	}
	options = append(options, additional...)
	var remote *checkedExporter
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	}
	if endpoint != "" {
		if err := validateOTLPEndpoint(endpoint); err != nil {
			return nil, err
		}
		// The official HTTP exporter honors the standard OTEL_* endpoint, TLS,
		// and header variables, including the traces-specific override.
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithTimeout(10*time.Second))
		if err != nil {
			return nil, fmt.Errorf("configure OTLP/HTTP exporter: %w", err)
		}
		remote = &checkedExporter{inner: exporter}
		options = append(options, sdktrace.WithBatcher(remote, sdktrace.WithBatchTimeout(time.Second)))
	}
	return &tracingSession{
		provider: sdktrace.NewTracerProvider(options...),
		remote:   remote,
	}, nil
}

func validateOTLPEndpoint(endpoint string) error {
	address, err := url.Parse(endpoint)
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") ||
		address.Hostname() == "" || address.User != nil || address.RawQuery != "" ||
		address.ForceQuery || address.Fragment != "" {
		return errors.New("OTLP/HTTP endpoint must be an http(s) URL without userinfo, a query, or a fragment")
	}
	return nil
}

func (t *tracingSession) Flush(ctx context.Context) error {
	err := t.provider.ForceFlush(ctx)
	if t.remote != nil {
		err = errors.Join(err, t.remote.Err())
	}
	return err
}

func (t *tracingSession) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	flushErr := t.Flush(ctx)
	shutdownErr := t.provider.Shutdown(ctx)
	if t.remote != nil {
		shutdownErr = errors.Join(shutdownErr, t.remote.Err())
	}
	return errors.Join(flushErr, shutdownErr)
}

// Batch exports can fail before ForceFlush is called. Remember those errors
// rather than allowing an asynchronous log message to become a false success.
// No global provider, propagator, or error handler is changed.
type checkedExporter struct {
	inner sdktrace.SpanExporter
	mu    sync.Mutex
	err   error
}

func (e *checkedExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	err := e.inner.ExportSpans(ctx, spans)
	e.remember(err)
	return err
}

func (e *checkedExporter) Shutdown(ctx context.Context) error {
	err := e.inner.Shutdown(ctx)
	e.remember(err)
	return err
}

func (e *checkedExporter) remember(err error) {
	if err != nil {
		e.mu.Lock()
		e.err = errors.Join(e.err, err)
		e.mu.Unlock()
	}
}

func (e *checkedExporter) Err() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.err
}

type tracedActivityContext struct {
	task.ActivityContext
	traced context.Context
}

func (c tracedActivityContext) Context() context.Context { return c.traced }

func traceActivity(tracer trace.Tracer, name, spanName string, activity task.Activity) task.Activity {
	return func(ctx task.ActivityContext) (result any, err error) {
		// DTS owns durable spans; create an application span under its context.
		traced, span := tracer.Start(ctx.Context(), spanName, trace.WithSpanKind(trace.SpanKindInternal))
		span.SetAttributes(attribute.String("sample.activity", name))
		defer func() {
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "activity failed")
			}
			span.End()
		}()
		return activity(tracedActivityContext{ActivityContext: ctx, traced: traced})
	}
}
