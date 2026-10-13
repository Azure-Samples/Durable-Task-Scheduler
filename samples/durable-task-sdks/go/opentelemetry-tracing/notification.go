package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	outboundSpanName = "app.notification_http"
	serverSpanName   = "app.notification_endpoint"
)

// The demo uses a loopback notification fixture rather than contacting customers.
func notificationServer(tracer trace.Tracer) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		_, span := tracer.Start(ctx, serverSpanName, trace.WithSpanKind(trace.SpanKindServer))
		span.End()
		w.WriteHeader(http.StatusNoContent)
	}))
}

func callNotification(ctx context.Context, tracer trace.Tracer, targetURL string) (err error) {
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
		return err
	}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(request.Header))
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("notification endpoint returned %s", response.Status)
	}
	return nil
}
