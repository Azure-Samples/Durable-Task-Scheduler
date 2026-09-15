package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
	"github.com/microsoft/durabletask-go/api"
	dts "github.com/microsoft/durabletask-go/durabletaskscheduler"
	"github.com/microsoft/durabletask-go/task"
)

const (
	orchestrationName = "GoBoundedCoordinator"
	childName         = "GoBoundedCoordinatorProcessItem"
	getBatchName      = "GoBoundedCoordinatorGetNextBatch"
	applyName         = "GoBoundedCoordinatorApplyChange"
	checkpointEvent   = "GoBoundedCoordinatorVerifyCheckpoint"
	carryoverEvent    = "GoBoundedCoordinatorCarryover"
	carryoverPayload  = "queued-before-first-history-reset"
	totalBatches      = 3
	itemsPerBatch     = 5
	batchLimit        = 5
	sourceLimit       = 50
)

type CoordinatorState struct {
	Cursor      string `json:"cursor"`
	BatchNumber int    `json:"batch_number"`
	Processed   int    `json:"processed"`
}

func cursorFor(batch int) string {
	if batch == 0 {
		return ""
	}
	return fmt.Sprintf("cursor-%d", batch)
}

func (state CoordinatorState) validate() error {
	if state.BatchNumber < 0 || state.BatchNumber >= totalBatches ||
		state.Cursor != cursorFor(state.BatchNumber) || state.Processed != state.BatchNumber*itemsPerBatch {
		return fmt.Errorf("invalid coordinator carry-forward state: %+v", state)
	}
	return nil
}

type BatchRequest struct {
	Cursor   string `json:"cursor"`
	MaxItems int    `json:"max_items"`
}

type Item struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Payload  string `json:"payload"`
}

type Batch struct {
	Items      []Item `json:"items"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

type Checkpoint struct {
	Phase       string `json:"phase"`
	BatchNumber int    `json:"batch_number"`
	Cursor      string `json:"cursor"`
	Processed   int    `json:"processed"`
}

type CoordinatorResult struct {
	TotalBatches int    `json:"total_batches"`
	Processed    int    `json:"processed"`
	Completed    bool   `json:"completed"`
	Carryover    string `json:"carryover"`
}

type ExecutionEvidence struct {
	BatchNumber       int    `json:"batch_number"`
	ExecutionID       string `json:"execution_id"`
	BatchActivities   int    `json:"batch_activities"`
	CompletedChildren int    `json:"completed_children"`
	CarryoverEvents   int    `json:"carryover_events"`
}

func nextBatch(input BatchRequest) (Batch, error) {
	if input.MaxItems < itemsPerBatch || input.MaxItems > sourceLimit {
		return Batch{}, fmt.Errorf("max_items must be between %d and %d; fixture cursors advance by whole pages",
			itemsPerBatch, sourceLimit)
	}
	previous := 0
	if input.Cursor != "" {
		raw, ok := strings.CutPrefix(input.Cursor, "cursor-")
		if !ok {
			return Batch{}, fmt.Errorf("invalid cursor %q", input.Cursor)
		}
		var err error
		previous, err = strconv.Atoi(raw)
		if err != nil || previous < 1 || previous > totalBatches || input.Cursor != cursorFor(previous) {
			return Batch{}, fmt.Errorf("invalid cursor %q", input.Cursor)
		}
	}
	if previous == totalBatches {
		return Batch{Items: []Item{}}, nil
	}
	batchNumber := previous + 1
	batch := Batch{
		Items:      make([]Item, itemsPerBatch),
		NextCursor: cursorFor(batchNumber), HasMore: batchNumber < totalBatches,
	}
	for i := range batch.Items {
		batch.Items[i] = Item{
			ID: fmt.Sprintf("item-%d-%d", batchNumber, i+1), TenantID: fmt.Sprintf("tenant-%d", i+1),
			Payload: fmt.Sprintf("data-%d-%d", batchNumber, i+1),
		}
	}
	return batch, nil
}

func getNextBatch(ctx task.ActivityContext) (any, error) {
	var input BatchRequest
	if err := ctx.GetInput(&input); err != nil {
		return nil, err
	}
	// Simulation only: the cursor addresses a stateless, finite source fixture.
	return nextBatch(input)
}

func applyChange(ctx task.ActivityContext) (any, error) {
	var item Item
	if err := ctx.GetInput(&item); err != nil {
		return nil, err
	}
	if item.ID == "" || item.TenantID == "" || item.Payload == "" {
		return nil, errors.New("change requires an item ID, tenant ID, and payload")
	}
	// Simulation only: no tenant data is changed.
	return "processed:" + item.ID, nil
}

func processItem(ctx *task.OrchestrationContext) (any, error) {
	var item Item
	if err := ctx.GetInput(&item); err != nil {
		return nil, err
	}
	var receipt string
	if err := ctx.CallActivity(applyName, task.WithActivityInput(item)).Await(&receipt); err != nil {
		return nil, fmt.Errorf("apply item %s: %w", item.ID, err)
	}
	return receipt, nil
}

func childID(parent api.InstanceID, itemID string) string {
	return string(parent) + "-" + itemID
}

func coordinator(ctx *task.OrchestrationContext) (any, error) {
	var state CoordinatorState
	if err := ctx.GetInput(&state); err != nil {
		return nil, err
	}
	if err := state.validate(); err != nil {
		return nil, err
	}
	var batch Batch
	if err := ctx.CallActivity(getBatchName, task.WithActivityInput(BatchRequest{
		Cursor: state.Cursor, MaxItems: batchLimit,
	})).Await(&batch); err != nil {
		return nil, fmt.Errorf("read bounded batch: %w", err)
	}
	if len(batch.Items) != itemsPerBatch || len(batch.Items) > batchLimit ||
		batch.NextCursor != cursorFor(state.BatchNumber+1) ||
		batch.HasMore != (state.BatchNumber+1 < totalBatches) {
		return nil, fmt.Errorf("unexpected source batch: %+v", batch)
	}

	pending := make([]task.Task, len(batch.Items))
	for i, item := range batch.Items {
		pending[i] = ctx.CallSubOrchestrator(childName,
			task.WithSubOrchestrationInstanceID(childID(ctx.ID, item.ID)),
			task.WithSubOrchestratorInput(item))
	}
	if err := ctx.WhenAll(pending...); err != nil {
		return nil, fmt.Errorf("drain child batch: %w", err)
	}
	for i, child := range pending {
		var receipt string
		if err := child.Await(&receipt); err != nil {
			return nil, fmt.Errorf("decode child receipt: %w", err)
		}
		if receipt != "processed:"+batch.Items[i].ID {
			return nil, fmt.Errorf("unexpected child receipt %q", receipt)
		}
	}
	state.BatchNumber++
	state.Processed += len(batch.Items)
	state.Cursor = batch.NextCursor

	// Demo-only checkpoint: let the client inspect this real execution before resetting it.
	if err := ctx.SetCustomStatusValue(Checkpoint{
		Phase: "awaiting-verification", BatchNumber: state.BatchNumber, Cursor: state.Cursor, Processed: state.Processed,
	}); err != nil {
		return nil, err
	}
	waitCtx, cancelWait := ctx.WithCancel()
	var acknowledgedBatch int
	err := waitCtx.WaitForSingleEvent(checkpointEvent, 15*time.Second).Await(&acknowledgedBatch)
	cancelWait()
	if err != nil {
		return nil, fmt.Errorf("verify batch %d checkpoint: %w", state.BatchNumber, err)
	}
	if acknowledgedBatch != state.BatchNumber {
		return nil, fmt.Errorf("checkpoint acknowledged batch %d, want %d", acknowledgedBatch, state.BatchNumber)
	}

	if batch.HasMore {
		ctx.ContinueAsNew(state, task.WithKeepUnprocessedEvents())
		return nil, nil
	}
	var carried string
	if err := ctx.WaitForSingleEvent(carryoverEvent, 0).Await(&carried); err != nil {
		return nil, fmt.Errorf("carryover event did not survive history resets: %w", err)
	}
	if carried != carryoverPayload {
		return nil, fmt.Errorf("incorrect carried event %q", carried)
	}
	return CoordinatorResult{
		TotalBatches: state.BatchNumber, Processed: state.Processed, Completed: true, Carryover: carried,
	}, nil
}

func newRegistry() (*task.TaskRegistry, error) {
	r := task.NewTaskRegistry()
	return r, errors.Join(
		r.AddOrchestratorN(orchestrationName, coordinator),
		r.AddOrchestratorN(childName, processItem),
		r.AddActivityN(getBatchName, getNextBatch),
		r.AddActivityN(applyName, applyChange),
	)
}

func verifyBatchHistory(history *api.OrchestrationHistory, parent api.InstanceID, batchNumber int) (ExecutionEvidence, error) {
	evidence := ExecutionEvidence{BatchNumber: batchNumber}
	if history == nil || history.ExecutionID == "" || history.InstanceID != parent {
		return evidence, errors.New("coordinator history has missing or incorrect execution identity")
	}
	evidence.ExecutionID = history.ExecutionID
	expectedItems := make(map[string]Item, itemsPerBatch)
	for i := 1; i <= itemsPerBatch; i++ {
		item := Item{
			ID: fmt.Sprintf("item-%d-%d", batchNumber, i), TenantID: fmt.Sprintf("tenant-%d", i),
			Payload: fmt.Sprintf("data-%d-%d", batchNumber, i),
		}
		expectedItems[item.ID] = item
	}
	children := make(map[int32]string, itemsPerBatch)
	finished := make(map[int32]bool, itemsPerBatch)
	seenItems := make(map[string]bool, itemsPerBatch)
	starts, batchCompletions := 0, 0
	for _, event := range history.Events {
		if event == nil {
			return evidence, errors.New("nil event in coordinator history")
		}
		switch event.Type {
		case api.HistoryEventExecutionStarted:
			starts++
			var state CoordinatorState
			if err := event.ReadInput(&state); err != nil {
				return evidence, err
			}
			want := CoordinatorState{Cursor: cursorFor(batchNumber - 1), BatchNumber: batchNumber - 1,
				Processed: (batchNumber - 1) * itemsPerBatch}
			if state != want {
				return evidence, fmt.Errorf("execution input = %+v, want %+v", state, want)
			}
		case api.HistoryEventTaskScheduled:
			evidence.BatchActivities++
			var input BatchRequest
			if err := event.ReadInput(&input); err != nil {
				return evidence, err
			}
			if event.TaskScheduled == nil || event.TaskScheduled.Name != getBatchName ||
				input != (BatchRequest{Cursor: cursorFor(batchNumber - 1), MaxItems: batchLimit}) {
				return evidence, fmt.Errorf("unexpected batch activity: %+v", event)
			}
		case api.HistoryEventTaskCompleted:
			batchCompletions++
		case api.HistoryEventSubOrchestrationInstanceCreated:
			var item Item
			if err := event.ReadInput(&item); err != nil {
				return evidence, err
			}
			created := event.SubOrchestrationInstanceCreated
			expected, exists := expectedItems[item.ID]
			if created == nil || created.Name != childName ||
				string(created.InstanceID) != childID(parent, item.ID) ||
				!exists || expected != item || seenItems[item.ID] {
				return evidence, fmt.Errorf("unexpected/duplicate child item: %+v", item)
			}
			if _, exists := children[event.EventID]; exists {
				return evidence, fmt.Errorf("duplicate child task ID %d", event.EventID)
			}
			children[event.EventID] = item.ID
			seenItems[item.ID] = true
		case api.HistoryEventSubOrchestrationInstanceCompleted:
			completed := event.SubOrchestrationInstanceCompleted
			if completed == nil {
				return evidence, errors.New("child completion has no details")
			}
			itemID, exists := children[completed.TaskScheduledID]
			if !exists || finished[completed.TaskScheduledID] {
				return evidence, errors.New("child completed without a unique creation event")
			}
			var receipt string
			if err := event.ReadResult(&receipt); err != nil {
				return evidence, err
			}
			if receipt != "processed:"+itemID {
				return evidence, fmt.Errorf("history child receipt = %q, want processed:%s", receipt, itemID)
			}
			finished[completed.TaskScheduledID] = true
			evidence.CompletedChildren++
		case api.HistoryEventEventRaised:
			if event.EventRaised != nil && strings.EqualFold(event.EventRaised.Name, carryoverEvent) {
				var payload string
				if err := event.ReadInput(&payload); err != nil {
					return evidence, err
				}
				if payload != carryoverPayload {
					return evidence, fmt.Errorf("unexpected carryover payload %q", payload)
				}
				evidence.CarryoverEvents++
			}
		}
	}
	if starts != 1 || evidence.BatchActivities != 1 || batchCompletions != 1 ||
		len(children) != itemsPerBatch || evidence.CompletedChildren != itemsPerBatch {
		return evidence, fmt.Errorf("batch history was not bounded/reset: %+v (starts=%d batch completions=%d children=%d)",
			evidence, starts, batchCompletions, len(children))
	}
	if evidence.CarryoverEvents > 1 || (batchNumber > 1 && evidence.CarryoverEvents != 1) {
		return evidence, fmt.Errorf("carryover event missing or duplicated in execution %d: %+v", batchNumber, evidence)
	}
	return evidence, nil
}

func waitForCheckpoint(ctx context.Context, c *dts.Client, id api.InstanceID, batch int) (*api.OrchestrationMetadata, error) {
	var metadata *api.OrchestrationMetadata
	err := sample.Until(ctx, 50*time.Millisecond, func() (bool, error) {
		var err error
		metadata, err = c.FetchOrchestrationMetadata(ctx, id, api.WithFetchPayloads(true))
		if err != nil {
			return false, err
		}
		if metadata.IsComplete() {
			return false, fmt.Errorf("coordinator ended before checkpoint %d: %s (%+v)",
				batch, metadata.RuntimeStatus, metadata.FailureDetails)
		}
		if metadata.SerializedCustomStatus == "" {
			return false, nil
		}
		var checkpoint Checkpoint
		if err := metadata.ReadCustomStatus(&checkpoint); err != nil {
			return false, err
		}
		if checkpoint.BatchNumber > batch {
			return false, fmt.Errorf("skipped checkpoint %d: %+v", batch, checkpoint)
		}
		if checkpoint.BatchNumber != batch {
			return false, nil
		}
		want := Checkpoint{
			Phase: "awaiting-verification", BatchNumber: batch, Cursor: cursorFor(batch), Processed: batch * itemsPerBatch,
		}
		return true, sample.Require(checkpoint == want, "checkpoint = %+v, want %+v", checkpoint, want)
	})
	return metadata, err
}

func run(ctx context.Context) error {
	r, err := newRegistry()
	if err != nil {
		return err
	}
	return sample.WithHost(ctx, r, func(ctx context.Context, c *dts.Client) (err error) {
		id, err := c.ScheduleNewOrchestration(ctx, orchestrationName,
			api.WithInstanceID(sample.ID("bounded-coordinator")), api.WithInput(CoordinatorState{}))
		if err != nil {
			return err
		}
		defer stopOnError(c, id, &err)

		executionIDs := make(map[string]bool, totalBatches)
		evidence := make([]ExecutionEvidence, 0, totalBatches)
		for batch := 1; batch <= totalBatches; batch++ {
			metadata, err := waitForCheckpoint(ctx, c, id, batch)
			if err != nil {
				return err
			}
			query := api.HistoryQuery{ExecutionID: metadata.ExecutionID, MaxEvents: 200}
			history, err := c.GetOrchestrationHistory(ctx, id, query)
			if err != nil {
				return fmt.Errorf("read real execution %d history: %w", batch, err)
			}
			current, err := verifyBatchHistory(history, id, batch)
			if err != nil {
				return err
			}
			if executionIDs[current.ExecutionID] {
				return fmt.Errorf("batch %d reused execution %s instead of continuing as new", batch, current.ExecutionID)
			}
			executionIDs[current.ExecutionID] = true

			if batch == 1 {
				if err := c.RaiseEvent(ctx, id, carryoverEvent, api.WithEventPayload(carryoverPayload)); err != nil {
					return err
				}
				// Observe the event in execution one before allowing either history reset.
				if err := sample.Until(ctx, 50*time.Millisecond, func() (bool, error) {
					history, err := c.GetOrchestrationHistory(ctx, id, query)
					if err != nil {
						return false, err
					}
					current, err = verifyBatchHistory(history, id, batch)
					return current.CarryoverEvents == 1, err
				}); err != nil {
					return err
				}
			}
			evidence = append(evidence, current)
			if err := sample.PrintJSON(current); err != nil {
				return err
			}
			if err := c.RaiseEvent(ctx, id, checkpointEvent, api.WithEventPayload(batch)); err != nil {
				return err
			}
		}
		var result CoordinatorResult
		if err := sample.Wait(ctx, c, id, &result); err != nil {
			return err
		}
		want := CoordinatorResult{TotalBatches: 3, Processed: 15, Completed: true, Carryover: carryoverPayload}
		if err := sample.Require(result == want && len(executionIDs) == 3,
			"coordinator result = %+v, want %+v across three executions", result, want); err != nil {
			return err
		}
		return sample.PrintJSON(struct {
			InstanceID api.InstanceID      `json:"instance_id"`
			Executions []ExecutionEvidence `json:"executions"`
			Result     CoordinatorResult   `json:"result"`
		}{id, evidence, result})
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
	sample.Main("bounded-coordinator", run)
}
