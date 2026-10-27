package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/microsoft/durabletask-go/api"
)

type activityInput []byte

func (input activityInput) GetInput(target any) error { return json.Unmarshal(input, target) }
func (activityInput) Context() context.Context        { return context.Background() }

func jsonValue(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestBoundedStatelessBatches(t *testing.T) {
	total := 0
	for batchNumber := 1; batchNumber <= totalBatches; batchNumber++ {
		input := BatchRequest{Cursor: cursorFor(batchNumber - 1), MaxItems: batchLimit}
		batch, err := nextBatch(input)
		if err != nil {
			t.Fatal(err)
		}
		repeated, err := nextBatch(input)
		if err != nil || !reflect.DeepEqual(batch, repeated) {
			t.Fatal("a repeated cursor did not produce the same batch")
		}
		if len(batch.Items) != 5 || batch.NextCursor != cursorFor(batchNumber) ||
			batch.HasMore != (batchNumber < totalBatches) {
			t.Fatalf("incorrect batch %d: %+v", batchNumber, batch)
		}
		for i, item := range batch.Items {
			want := Item{ID: fmt.Sprintf("item-%d-%d", batchNumber, i+1), TenantID: fmt.Sprintf("tenant-%d", i+1),
				Payload: fmt.Sprintf("data-%d-%d", batchNumber, i+1)}
			if item != want {
				t.Fatalf("item = %+v, want %+v", item, want)
			}
			output, err := applyChange(activityInput(jsonValue(t, item)))
			if err != nil || output != "processed:"+item.ID {
				t.Fatalf("receipt = %v, %v", output, err)
			}
		}
		total += len(batch.Items)
	}
	if total != 15 {
		t.Fatalf("processed %d items, want 15", total)
	}
	exhausted, err := nextBatch(BatchRequest{Cursor: "cursor-3", MaxItems: batchLimit})
	if err != nil || len(exhausted.Items) != 0 || exhausted.HasMore || exhausted.NextCursor != "" {
		t.Fatalf("exhausted source = %+v, %v", exhausted, err)
	}
	larger, err := nextBatch(BatchRequest{MaxItems: sourceLimit})
	if err != nil || len(larger.Items) != itemsPerBatch {
		t.Fatalf("source exceeded its page bound: %+v, %v", larger, err)
	}
}

func TestInvalidSourceAndState(t *testing.T) {
	for _, cursor := range []string{"invalid", "cursor-0", "cursor-01", "cursor-4", "cursor--1"} {
		if _, err := nextBatch(BatchRequest{Cursor: cursor, MaxItems: 5}); err == nil {
			t.Fatalf("invalid cursor accepted: %s", cursor)
		}
	}
	for _, limit := range []int{-1, 0, 1, itemsPerBatch - 1, sourceLimit + 1} {
		if _, err := nextBatch(BatchRequest{MaxItems: limit}); err == nil {
			t.Fatalf("invalid source bound accepted: %d", limit)
		}
	}
	for _, state := range []CoordinatorState{
		{BatchNumber: -1}, {BatchNumber: 1}, {Cursor: "cursor-1", BatchNumber: 1, Processed: 4},
		{Cursor: "cursor-3", BatchNumber: 3, Processed: 15},
	} {
		if err := state.validate(); err == nil {
			t.Fatalf("invalid carry-forward state accepted: %+v", state)
		}
	}
	if _, err := applyChange(activityInput(`{"id":"item-1"}`)); err == nil {
		t.Fatal("incomplete tenant change accepted")
	}
	if _, err := getNextBatch(activityInput(`{`)); err == nil {
		t.Fatal("malformed source request accepted")
	}
}

func historyForBatch(t *testing.T, batch int) *api.OrchestrationHistory {
	t.Helper()
	const parent api.InstanceID = "go-bounded-test"
	history := &api.OrchestrationHistory{InstanceID: parent, ExecutionID: fmt.Sprintf("execution-%d", batch)}
	history.Events = append(history.Events,
		&api.HistoryEvent{Type: api.HistoryEventExecutionStarted, ExecutionStarted: &api.HistoryExecutionStartedEvent{
			SerializedInput: jsonValue(t, CoordinatorState{
				Cursor: cursorFor(batch - 1), BatchNumber: batch - 1, Processed: (batch - 1) * 5,
			}),
		}},
		&api.HistoryEvent{Type: api.HistoryEventTaskScheduled, TaskScheduled: &api.HistoryTaskScheduledEvent{
			Name: getBatchName, SerializedInput: jsonValue(t, BatchRequest{Cursor: cursorFor(batch - 1), MaxItems: 5}),
		}},
		&api.HistoryEvent{Type: api.HistoryEventTaskCompleted},
	)
	for i := 1; i <= itemsPerBatch; i++ {
		item := Item{ID: fmt.Sprintf("item-%d-%d", batch, i), TenantID: fmt.Sprintf("tenant-%d", i),
			Payload: fmt.Sprintf("data-%d-%d", batch, i)}
		history.Events = append(history.Events,
			&api.HistoryEvent{Type: api.HistoryEventSubOrchestrationInstanceCreated, EventID: int32(i),
				SubOrchestrationInstanceCreated: &api.HistorySubOrchestrationInstanceCreatedEvent{
					InstanceID: api.InstanceID(childID(parent, item.ID)), Name: childName, SerializedInput: jsonValue(t, item),
				}},
			&api.HistoryEvent{Type: api.HistoryEventSubOrchestrationInstanceCompleted,
				SubOrchestrationInstanceCompleted: &api.HistoryTaskResultEvent{
					TaskScheduledID: int32(i), SerializedResult: jsonValue(t, "processed:"+item.ID),
				}},
		)
	}
	history.Events = append(history.Events, &api.HistoryEvent{
		Type: api.HistoryEventEventRaised, EventRaised: &api.HistoryExternalEvent{
			Name: carryoverEvent, SerializedInput: jsonValue(t, carryoverPayload),
		},
	})
	return history
}

func TestExecutionEvidence(t *testing.T) {
	for batch := 1; batch <= totalBatches; batch++ {
		history := historyForBatch(t, batch)
		evidence, err := verifyBatchHistory(history, history.InstanceID, batch)
		if err != nil || evidence.CompletedChildren != 5 || evidence.BatchActivities != 1 ||
			evidence.CarryoverEvents != 1 || evidence.ExecutionID != fmt.Sprintf("execution-%d", batch) {
			t.Fatalf("evidence = %+v, %v", evidence, err)
		}
	}
	for _, mutate := range []func(*api.OrchestrationHistory){
		func(h *api.OrchestrationHistory) { h.ExecutionID = "" },
		func(h *api.OrchestrationHistory) { h.Events = h.Events[:len(h.Events)-1] },
		func(h *api.OrchestrationHistory) { h.Events = append(h.Events, h.Events[1]) },
		func(h *api.OrchestrationHistory) { h.Events = append(h.Events, h.Events[3]) },
		func(h *api.OrchestrationHistory) { h.Events = append(h.Events, nil) },
		func(h *api.OrchestrationHistory) {
			h.Events[4].SubOrchestrationInstanceCompleted.SerializedResult = `"wrong"`
		},
		func(h *api.OrchestrationHistory) { h.Events[0].ExecutionStarted.SerializedInput = `{}` },
	} {
		history := historyForBatch(t, 2)
		mutate(history)
		if _, err := verifyBatchHistory(history, history.InstanceID, 2); err == nil {
			t.Fatal("invalid reset/child/carryover evidence accepted")
		}
	}
}
