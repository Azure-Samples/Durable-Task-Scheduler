package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoEternalPeriodicCleanup"
	cleanupName       = "GoEternalCleanupTask"
	iterations        = 5
	cleanupInterval   = 250 * time.Millisecond
)

type CleanupState struct {
	Iteration    int `json:"iteration"`
	TotalRemoved int `json:"total_removed"`
}

func (state CleanupState) validate() error {
	if state.Iteration < 1 || state.Iteration > iterations || state.TotalRemoved < 0 {
		return fmt.Errorf("invalid cleanup carry-forward state: %+v", state)
	}
	return nil
}

type CleanupReceipt struct {
	Iteration int      `json:"iteration"`
	Removed   []string `json:"removed"`
	Retained  []string `json:"retained"`
	Message   string   `json:"message"`
}

type CleanupResult struct {
	Iterations   int    `json:"iterations"`
	TotalRemoved int    `json:"total_removed"`
	LastMessage  string `json:"last_message"`
}

func cleanupFixture(iteration int) (CleanupReceipt, error) {
	if iteration < 1 || iteration > iterations {
		return CleanupReceipt{}, errors.New("cleanup iteration is outside the fixture")
	}
	// Simulation only: partition in-memory records; never delete user files or data.
	records := []struct {
		id      string
		expired bool
	}{
		{fmt.Sprintf("expired-%d-a", iteration), true},
		{fmt.Sprintf("current-%d", iteration), false},
		{fmt.Sprintf("expired-%d-b", iteration), true},
	}
	receipt := CleanupReceipt{Iteration: iteration, Message: "Cleanup completed"}
	for _, record := range records {
		if record.expired {
			receipt.Removed = append(receipt.Removed, record.id)
		} else {
			receipt.Retained = append(receipt.Retained, record.id)
		}
	}
	return receipt, nil
}

func cleanupTask(ctx task.ActivityContext) (any, error) {
	var iteration int
	if err := ctx.GetInput(&iteration); err != nil {
		return nil, err
	}
	return cleanupFixture(iteration)
}

func periodicCleanup(ctx *task.OrchestrationContext) (any, error) {
	var state CleanupState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := state.validate(); err != nil {
		return nil, err
	}
	var receipt CleanupReceipt
	if err := ctx.CallActivity(cleanupName, task.WithActivityInput(state.Iteration)).Await(&receipt); err != nil {
		return nil, fmt.Errorf("cleanup cycle %d: %w", state.Iteration, err)
	}
	if receipt.Iteration != state.Iteration || receipt.Message != "Cleanup completed" {
		return nil, fmt.Errorf("invalid cleanup receipt: %+v", receipt)
	}
	state.TotalRemoved += len(receipt.Removed)
	if err := ctx.SetCustomStatusValue(state); err != nil {
		return nil, err
	}
	if err := ctx.CreateTimer(cleanupInterval).Await(nil); err != nil {
		return nil, fmt.Errorf("cleanup interval: %w", err)
	}
	if state.Iteration == iterations {
		return CleanupResult{
			Iterations: state.Iteration, TotalRemoved: state.TotalRemoved, LastMessage: receipt.Message,
		}, nil
	}

	state.Iteration++
	ctx.ContinueAsNew(state, task.WithKeepUnprocessedEvents())
	return nil, nil
}

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, periodicCleanup),
		r.AddActivityN(cleanupName, cleanupTask),
	)
}

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
	return sample.Require(starts == 1 && scheduled == 1 && completed == 1 && timers == 1 && fired == 1,
		"latest history was not reset: starts=%d activities=%d/%d timers=%d/%d",
		starts, scheduled, completed, timers, fired)
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
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
		if err := sample.Require(result == want, "cleanup result = %+v, want %+v", result, want); err != nil {
			return err
		}
		history, err := c.GetOrchestrationHistory(ctx, id, api.HistoryQuery{MaxEvents: 100})
		if err != nil {
			return fmt.Errorf("verify cleanup history reset: %w", err)
		}
		if err := verifyLatestHistory(history); err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID              api.InstanceID `json:"instance_id"`
			FinalExecutionID        string         `json:"final_execution_id"`
			LatestCleanupActivities int            `json:"latest_cleanup_activities"`
			Result                  CleanupResult  `json:"result"`
		}{id, history.ExecutionID, 1, result})
	})
}

func stopOnError(c *dts.Client, id api.InstanceID, runErr *error) {
	if *runErr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, err := c.FetchOrchestrationMetadata(ctx, id)
	if err == nil && !state.IsComplete() {
		err = c.TerminateOrchestration(ctx, id)
		if err == nil {
			_, err = c.WaitForOrchestrationCompletion(ctx, id)
		}
	}
	*runErr = errors.Join(*runErr, err)
}

func main() {
	sample.Main("eternal-orchestrations", run)
}
