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

func TestCleanupFixture(t *testing.T) {
	removed := 0
	for iteration := 1; iteration <= iterations; iteration++ {
		got, err := cleanupTask(activityInput(fmt.Sprint(iteration)))
		if err != nil {
			t.Fatal(err)
		}
		want := CleanupReceipt{
			Iteration: iteration,
			Removed:   []string{fmt.Sprintf("expired-%d-a", iteration), fmt.Sprintf("expired-%d-b", iteration)},
			Retained:  []string{fmt.Sprintf("current-%d", iteration)}, Message: "Cleanup completed",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cleanup = %+v, want %+v", got, want)
		}
		removed += len(got.(CleanupReceipt).Removed)
	}
	if removed != 10 {
		t.Fatalf("removed %d records, want 10", removed)
	}
	for _, input := range []activityInput{[]byte(`0`), []byte(`6`), []byte(`"wrong"`)} {
		if _, err := cleanupTask(input); err == nil {
			t.Fatalf("invalid iteration accepted: %s", input)
		}
	}
}

func validHistory() *api.OrchestrationHistory {
	return &api.OrchestrationHistory{
		ExecutionID: "execution-5",
		Events: []*api.HistoryEvent{
			{Type: api.HistoryEventExecutionStarted, ExecutionStarted: &api.HistoryExecutionStartedEvent{
				SerializedInput: `{"iteration":5,"total_removed":8}`,
			}},
			{Type: api.HistoryEventTaskScheduled, TaskScheduled: &api.HistoryTaskScheduledEvent{
				Name: cleanupName, SerializedInput: `5`,
			}},
			{Type: api.HistoryEventTaskCompleted, TaskCompleted: &api.HistoryTaskResultEvent{
				SerializedResult: `{"iteration":5,"removed":["expired-5-a","expired-5-b"],"retained":["current-5"],"message":"Cleanup completed"}`,
			}},
			{Type: api.HistoryEventTimerCreated},
			{Type: api.HistoryEventTimerFired},
		},
	}
}

func TestHistoryResetEvidence(t *testing.T) {
	if err := verifyLatestHistory(validHistory()); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*api.OrchestrationHistory){
		func(h *api.OrchestrationHistory) { h.ExecutionID = "" },
		func(h *api.OrchestrationHistory) { h.Events = append(h.Events, h.Events[1]) },
		func(h *api.OrchestrationHistory) { h.Events = h.Events[:4] },
		func(h *api.OrchestrationHistory) {
			h.Events[0].ExecutionStarted.SerializedInput = `{"iteration":4,"total_removed":6}`
		},
		func(h *api.OrchestrationHistory) { h.Events[1].TaskScheduled.SerializedInput = `1` },
	} {
		history := validHistory()
		mutate(history)
		if err := verifyLatestHistory(history); err == nil {
			t.Fatal("incorrect reset evidence accepted")
		}
	}
}

func TestInvalidCarryForwardState(t *testing.T) {
	for _, state := range []CleanupState{{}, {Iteration: 6}, {Iteration: 1, TotalRemoved: -1}} {
		if err := state.validate(); err == nil {
			t.Fatalf("invalid carry-forward state accepted: %+v", state)
		}
	}
}
