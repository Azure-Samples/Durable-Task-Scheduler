package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/testutil"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
)

func verifyLatestHistory(history *api.OrchestrationHistory) error {
	if history == nil || history.ExecutionID == "" {
		return errors.New("latest cleanup execution has no execution ID")
	}
	var starts, scheduled, completed, timers, fired int
	for _, event := range history.Events {
		if event == nil {
			return errors.New("nil event in cleanup history")
		}
		switch event.Type {
		case api.HistoryEventExecutionStarted:
			starts++
			var state CleanupState
			if err := event.ReadInput(&state); err != nil {
				return err
			}
			want := CleanupState{Iteration: 5, TotalRemoved: 8}
			if state != want {
				return fmt.Errorf("latest execution input = %+v, want %+v", state, want)
			}
		case api.HistoryEventTaskScheduled:
			scheduled++
			var iteration int
			if err := event.ReadInput(&iteration); err != nil {
				return err
			}
			if event.TaskScheduled == nil || event.TaskScheduled.Name != cleanupName || iteration != 5 {
				return fmt.Errorf("unexpected activity in latest cleanup history: %+v", event)
			}
		case api.HistoryEventTaskCompleted:
			completed++
			var receipt CleanupReceipt
			if err := event.ReadResult(&receipt); err != nil {
				return err
			}
			want := CleanupReceipt{
				Iteration: 5, Removed: []string{"expired-5-a", "expired-5-b"},
				Retained: []string{"current-5"}, Message: "Cleanup completed",
			}
			if !reflect.DeepEqual(receipt, want) {
				return fmt.Errorf("last cleanup receipt = %+v, want %+v", receipt, want)
			}
		case api.HistoryEventTimerCreated:
			timers++
		case api.HistoryEventTimerFired:
			fired++
		}
	}
	return testutil.Require(starts == 1 && scheduled == 1 && completed == 1 && timers == 1 && fired == 1,
		"latest history was not reset: starts=%d activities=%d/%d timers=%d/%d",
		starts, scheduled, completed, timers, fired)
}

func TestIntegration(t *testing.T) {
	ctx := testutil.IntegrationContext(t)
	r, err := newRegistry()
	if err != nil {
		t.Fatal(err)
	}
	err = sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("eternal-cleanup")), api.WithInput(CleanupState{Iteration: 1}))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		var result CleanupResult
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		want := CleanupResult{Iterations: 5, TotalRemoved: 10, LastMessage: "Cleanup completed"}
		if err := testutil.Require(result == want, "cleanup result = %+v, want %+v", result, want); err != nil {
			return err
		}
		history, err := c.GetOrchestrationHistory(ctx, id, api.HistoryQuery{MaxEvents: 100})
		if err != nil {
			return fmt.Errorf("verify cleanup history reset: %w", err)
		}
		return verifyLatestHistory(history)
	})
	if err != nil {
		t.Fatal(err)
	}
}
