package main

import (
	"encoding/json"
	"testing"

	"github.com/microsoft/durabletask-go/api"
)

func orderHistoryFixture(t *testing.T, input string) *api.OrchestrationHistory {
	t.Helper()
	serialize := func(value string) string {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	history := &api.OrchestrationHistory{Events: []*api.HistoryEvent{{
		Type:             api.HistoryEventExecutionStarted,
		ExecutionStarted: &api.HistoryExecutionStartedEvent{SerializedInput: serialize(input)},
	}}}
	for i, step := range orderSteps {
		id := int32(i)
		history.Events = append(history.Events, &api.HistoryEvent{
			Type: api.HistoryEventTaskScheduled, EventID: id,
			TaskScheduled: &api.HistoryTaskScheduledEvent{Name: step.Activity, SerializedInput: serialize(input)},
		})
		input = step.Result + "(" + input + ")"
		history.Events = append(history.Events, &api.HistoryEvent{
			Type:          api.HistoryEventTaskCompleted,
			TaskCompleted: &api.HistoryTaskResultEvent{TaskScheduledID: id, SerializedResult: serialize(input)},
		})
	}
	history.Events = append(history.Events, &api.HistoryEvent{
		Type: api.HistoryEventExecutionCompleted,
		ExecutionCompleted: &api.HistoryExecutionCompletedEvent{
			RuntimeStatus: api.RUNTIME_STATUS_COMPLETED, SerializedResult: serialize(input),
		},
	})
	return history
}

func TestOrderHistoryChecksIntermediateAndTerminalResults(t *testing.T) {
	const order = "Order-12345"
	if err := verifyOrderHistory(orderHistoryFixture(t, order), order); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*api.OrchestrationHistory){
		func(h *api.OrchestrationHistory) { h.Events = h.Events[:len(h.Events)-1] },
		func(h *api.OrchestrationHistory) { h.Events[1].TaskScheduled.Name = "different activity" },
		func(h *api.OrchestrationHistory) { h.Events[1].TaskScheduled.SerializedInput = `"wrong order"` },
		func(h *api.OrchestrationHistory) { h.Events[2].TaskCompleted.TaskScheduledID = 99 },
		func(h *api.OrchestrationHistory) { h.Events[2].TaskCompleted.SerializedResult = `"wrong result"` },
		func(h *api.OrchestrationHistory) {
			h.Events[len(h.Events)-1].ExecutionCompleted.RuntimeStatus = api.RUNTIME_STATUS_FAILED
		},
	} {
		history := orderHistoryFixture(t, order)
		mutate(history)
		if err := verifyOrderHistory(history, order); err == nil {
			t.Fatal("corrupt/incomplete order history was accepted")
		}
	}
}
