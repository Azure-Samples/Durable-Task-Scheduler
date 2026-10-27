package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/microsoft/durabletask-go/api"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type orderStep struct {
	Activity string
	Span     string
	Result   string
}

var orderSteps = []orderStep{
	{validateOrderName, "app.validate_order", "Validated"},
	{processPaymentName, "app.process_payment", "Paid"},
	{shipOrderName, "app.ship_order", "Shipped"},
	{sendNotificationName, "app.send_notification", "Notified"},
}

func expectedOrderResult(input string) string {
	for _, step := range orderSteps {
		input = step.Result + "(" + input + ")"
	}
	return input
}

func verifyOrderHistory(history *api.OrchestrationHistory, input string) error {
	if history == nil {
		return errors.New("missing order history")
	}
	expected := make(map[int32]string)
	completed := make(map[int32]bool)
	scheduled, terminal := 0, 0
	value := input
	for _, event := range history.Events {
		if event == nil {
			return errors.New("nil order history event")
		}
		switch event.Type {
		case api.HistoryEventExecutionStarted:
			if event.ExecutionStarted == nil || !serializedStringEquals(event.ExecutionStarted.SerializedInput, input) {
				return errors.New("history contains an unexpected order input")
			}
		case api.HistoryEventTaskScheduled:
			if scheduled >= len(orderSteps) || event.TaskScheduled == nil ||
				event.TaskScheduled.Name != orderSteps[scheduled].Activity ||
				!serializedStringEquals(event.TaskScheduled.SerializedInput, value) {
				return errors.New("history contains an unexpected activity order/input")
			}
			value = orderSteps[scheduled].Result + "(" + value + ")"
			expected[event.EventID] = value
			scheduled++
		case api.HistoryEventTaskCompleted:
			if event.TaskCompleted == nil {
				return errors.New("missing activity result")
			}
			id := event.TaskCompleted.TaskScheduledID
			want, found := expected[id]
			if !found || completed[id] || !serializedStringEquals(event.TaskCompleted.SerializedResult, want) {
				return errors.New("history activity result/correlation mismatch")
			}
			completed[id] = true
		case api.HistoryEventExecutionCompleted:
			terminal++
			if event.ExecutionCompleted == nil || event.ExecutionCompleted.RuntimeStatus != api.RUNTIME_STATUS_COMPLETED ||
				!serializedStringEquals(event.ExecutionCompleted.SerializedResult, expectedOrderResult(input)) {
				return errors.New("history terminal result/status mismatch")
			}
		case api.HistoryEventTaskFailed, api.HistoryEventExecutionTerminated:
			return errors.New("order history contains a failure")
		}
	}
	if scheduled != len(orderSteps) || len(completed) != len(orderSteps) || terminal != 1 {
		return errors.New("history omits an activity or terminal result")
	}
	return nil
}

func serializedStringEquals(value, expected string) bool {
	var decoded string
	return json.Unmarshal([]byte(value), &decoded) == nil && decoded == expected
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

func verifyRestoredContext(observation spanObservation, caller trace.SpanContext) error {
	parent := observation.Parent
	if !parent.IsValid() || !parent.IsSampled() || !parent.IsRemote() ||
		observation.ParentRecording || parent.TraceID() != caller.TraceID() {
		return errors.New("activity did not receive the sampled, non-recording remote DTS context")
	}
	return nil
}

func verifyApplicationSpans(spans tracetest.SpanStubs, caller trace.SpanContext, observed map[string][]spanObservation) error {
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
	for _, step := range orderSteps {
		observations := observed[step.Span]
		if len(observations) != 1 {
			return fmt.Errorf("expected one activity invocation for %s, got %d", step.Activity, len(observations))
		}
		evidence := observations[0]
		if err := verifyRestoredContext(evidence, caller); err != nil {
			return err
		}
		span, ok := byID[evidence.Started.SpanID().String()]
		if !ok || span.Name != step.Span || span.SpanKind != trace.SpanKindInternal ||
			!span.Parent.IsRemote() || span.Parent.TraceID() != caller.TraceID() ||
			span.Parent.SpanID() != evidence.Parent.SpanID() || span.SpanContext.SpanID() == span.Parent.SpanID() {
			return fmt.Errorf("missing user span or wrong remote parent for %s", step.Activity)
		}
	}
	if len(observed[outboundSpanName]) != 1 || len(observed[serverSpanName]) != 1 {
		return errors.New("missing outbound HTTP or server observation")
	}
	notification := observed[orderSteps[len(orderSteps)-1].Span][0]
	outbound, ok := byID[observed[outboundSpanName][0].Started.SpanID().String()]
	if !ok || outbound.Name != outboundSpanName || outbound.SpanKind != trace.SpanKindClient ||
		outbound.Parent.SpanID() != notification.Started.SpanID() {
		return errors.New("outbound HTTP span is not a child of the notification user span")
	}
	serverObservation := observed[serverSpanName][0]
	if err := verifyRestoredContext(serverObservation, caller); err != nil {
		return err
	}
	server, ok := byID[serverObservation.Started.SpanID().String()]
	if !ok || server.Name != serverSpanName || server.SpanKind != trace.SpanKindServer ||
		!server.Parent.IsRemote() || server.Parent.SpanID() != outbound.SpanContext.SpanID() {
		return errors.New("HTTP server span did not receive the outbound span as its remote parent")
	}
	return nil
}
