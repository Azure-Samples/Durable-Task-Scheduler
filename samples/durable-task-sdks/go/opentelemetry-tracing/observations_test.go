package main

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type spanObservation struct {
	Parent          trace.SpanContext
	ParentRecording bool
	Started         trace.SpanContext
}

// Observe contexts supplied to the real production instrumentation without
// replacing any workflow, activity, exporter, or HTTP handler.
type observingTracer struct {
	trace.Tracer
	mu       sync.Mutex
	observed map[string][]spanObservation
}

func newObservingTracer(tracer trace.Tracer) *observingTracer {
	return &observingTracer{Tracer: tracer, observed: make(map[string][]spanObservation)}
}

func (t *observingTracer) Start(ctx context.Context, name string, options ...trace.SpanStartOption) (context.Context, trace.Span) {
	parent := trace.SpanFromContext(ctx)
	observed := spanObservation{Parent: parent.SpanContext(), ParentRecording: parent.IsRecording()}
	traced, span := t.Tracer.Start(ctx, name, options...)
	observed.Started = span.SpanContext()
	t.mu.Lock()
	t.observed[name] = append(t.observed[name], observed)
	t.mu.Unlock()
	return traced, span
}

func (t *observingTracer) snapshot() map[string][]spanObservation {
	t.mu.Lock()
	defer t.mu.Unlock()
	snapshot := make(map[string][]spanObservation, len(t.observed))
	for name, observations := range t.observed {
		snapshot[name] = append([]spanObservation(nil), observations...)
	}
	return snapshot
}

func callerContext(spans tracetest.SpanStubs, traceID string) (trace.SpanContext, error) {
	for _, span := range spans {
		if span.Name == callerSpanName && span.SpanContext.TraceID().String() == traceID &&
			span.SpanContext.IsValid() && span.SpanContext.IsSampled() {
			return span.SpanContext, nil
		}
	}
	return trace.SpanContext{}, errors.New("exporter did not receive a valid sampled caller span")
}
