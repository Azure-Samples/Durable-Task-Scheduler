package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/api"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func verifyOrderResult(result orderResult, input, traceID string) error {
	if len(result.Steps) != len(orderSteps) {
		return errors.New("order result omitted activity evidence")
	}
	value := input
	for i, step := range orderSteps {
		value = step.Result + "(" + value + ")"
		evidence := result.Steps[i]
		parent, parentErr := trace.SpanIDFromHex(evidence.ParentSpanID)
		span, spanErr := trace.SpanIDFromHex(evidence.SpanID)
		if evidence.Value != value || evidence.TraceID != traceID || parentErr != nil || !parent.IsValid() ||
			spanErr != nil || !span.IsValid() || span == parent {
			return fmt.Errorf("activity %s output or propagated trace evidence mismatch", step.Activity)
		}
		if i != len(orderSteps)-1 && evidence.Notification != nil {
			return errors.New("unexpected notification in a non-notification activity")
		}
	}
	receipt := result.Steps[len(orderSteps)-1].Notification
	if result.Value != value || receipt == nil || receipt.TraceID != traceID || !receipt.Sampled {
		return errors.New("final order result or HTTP trace evidence mismatch")
	}
	return nil
}

func persistedContext(value *api.HistoryTraceContext) trace.SpanContext {
	if value == nil {
		return trace.SpanContext{}
	}
	ctx := propagation.TraceContext{}.Extract(context.Background(), propagation.MapCarrier{
		"traceparent": value.TraceParent,
		"tracestate":  value.TraceState,
	})
	return trace.SpanContextFromContext(ctx)
}

func verifyHistoryTrace(history *api.OrchestrationHistory, id api.InstanceID, caller trace.SpanContext) error {
	if history == nil || history.InstanceID != id || history.ExecutionID == "" {
		return errors.New("missing target orchestration history")
	}
	started := 0
	scheduled := make(map[string]int)
	for _, event := range history.Events {
		if event == nil {
			return errors.New("nil history event")
		}
		switch event.Type {
		case api.HistoryEventExecutionStarted:
			started++
			if event.ExecutionStarted == nil || event.ExecutionStarted.Name != orchestratorName ||
				event.ExecutionStarted.InstanceID != id {
				return errors.New("trace history belongs to a different orchestration")
			}
			parent := persistedContext(event.ExecutionStarted.ParentTraceContext)
			if !parent.IsValid() || !parent.IsSampled() || parent.TraceID() != caller.TraceID() ||
				parent.SpanID() != caller.SpanID() {
				return errors.New("DTS history did not persist the sampled caller context")
			}
		case api.HistoryEventTaskScheduled:
			if event.TaskScheduled == nil {
				return errors.New("missing task schedule details")
			}
			parent := persistedContext(event.TaskScheduled.ParentTraceContext)
			if !parent.IsValid() || !parent.IsSampled() || parent.TraceID() != caller.TraceID() {
				return errors.New("DTS activity history did not preserve the sampled caller trace")
			}
			scheduled[event.TaskScheduled.Name]++
		}
	}
	if started != 1 || len(scheduled) != len(orderSteps) {
		return errors.New("history is missing the order's execution/activity trace contexts")
	}
	for _, step := range orderSteps {
		if scheduled[step.Activity] != 1 {
			return fmt.Errorf("history must contain exactly one %s schedule", step.Activity)
		}
	}
	return nil
}

func verifyApplicationSpans(spans tracetest.SpanStubs, caller trace.SpanContext, result orderResult) error {
	byID := make(map[string]tracetest.SpanStub)
	for _, span := range spans {
		if span.SpanContext.TraceID() == caller.TraceID() {
			if !span.SpanContext.IsValid() || !span.SpanContext.IsSampled() || span.EndTime.IsZero() ||
				span.Status.Code == codes.Error {
				return errors.New("application exported an invalid, unfinished, unsampled, or failed span")
			}
			byID[span.SpanContext.SpanID().String()] = span
		}
	}
	callerSpan, ok := byID[caller.SpanID().String()]
	if !ok || callerSpan.Name != callerSpanName || callerSpan.SpanKind != trace.SpanKindClient {
		return errors.New("in-memory exporter did not receive this caller span")
	}
	if len(result.Steps) != len(orderSteps) {
		return errors.New("missing activity span evidence")
	}
	for i, evidence := range result.Steps {
		span, ok := byID[evidence.SpanID]
		if !ok || span.Name != orderSteps[i].Span || span.SpanKind != trace.SpanKindInternal ||
			!span.Parent.IsRemote() || span.Parent.TraceID() != caller.TraceID() ||
			span.Parent.SpanID().String() != evidence.ParentSpanID {
			return fmt.Errorf("missing user span or wrong remote parent for %s", orderSteps[i].Activity)
		}
	}
	notification := result.Steps[len(result.Steps)-1]
	if notification.Notification == nil {
		return errors.New("missing notification receipt")
	}
	receipt := notification.Notification
	outbound, ok := byID[receipt.ParentSpanID]
	if !ok || outbound.Name != outboundSpanName || outbound.SpanKind != trace.SpanKindClient ||
		outbound.Parent.SpanID().String() != notification.SpanID {
		return errors.New("outbound HTTP span is not a child of the notification user span")
	}
	server, ok := byID[receipt.SpanID]
	if !ok || server.Name != serverSpanName || server.SpanKind != trace.SpanKindServer ||
		!server.Parent.IsRemote() || server.Parent.SpanID() != outbound.SpanContext.SpanID() {
		return errors.New("HTTP server span did not receive the outbound span as its remote parent")
	}
	return nil
}
